package grpc

import (
	"context"
	invClient "ecommerce-microservices/order-service/internal/client"
	"ecommerce-microservices/order-service/internal/domain"
	repo "ecommerce-microservices/order-service/internal/repository"
	pb "ecommerce-microservices/order-service/pb"
	"errors"
	"log"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type OrderServer struct {
	pb.UnimplementedOrderServiceServer
	orderStore      *repo.MongoOrderStore
	inventoryClient invClient.InventoryClient
}

func NewOrderServer(os *repo.MongoOrderStore, ic invClient.InventoryClient) *OrderServer {
	if os == nil {
		log.Fatalf("MongoOrderStore cannot be nil")
	}
	if ic == nil {
		log.Fatalf("InventoryClient cannot be nil")
	}
	return &OrderServer{
		orderStore:      os,
		inventoryClient: ic,
	}
}

func (s *OrderServer) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.OrderResponse, error) {
	log.Printf("Received CreateOrder request for user %s", req.UserId)
	if req.UserId == "" || len(req.Items) == 0 {
		log.Printf("CreateOrder validation failed: UserID or Items empty")
		return nil, status.Error(codes.InvalidArgument, "UserID and at least one item are required")
	}

	var orderItems []domain.OrderItem
	var totalAmount float64
	productIDs := make(map[string]bool)

	for _, itemInput := range req.Items {
		if itemInput.ProductId == "" || itemInput.Quantity <= 0 {
			log.Printf("CreateOrder validation failed: Invalid item data: ProductID '%s', Quantity %d", itemInput.ProductId, itemInput.Quantity)
			return nil, status.Errorf(codes.InvalidArgument, "Invalid item data: ProductID '%s', Quantity %d", itemInput.ProductId, itemInput.Quantity)
		}
		if _, exists := productIDs[itemInput.ProductId]; exists {
			log.Printf("CreateOrder validation failed: Duplicate product ID %s", itemInput.ProductId)
			return nil, status.Errorf(codes.InvalidArgument, "Duplicate product ID in order: %s", itemInput.ProductId)
		}
		productIDs[itemInput.ProductId] = true

		log.Printf("Calling Inventory Service for product data (price): %s", itemInput.ProductId)
		productInfo, err := s.inventoryClient.GetProduct(ctx, itemInput.ProductId)
		if err != nil {
			log.Printf("Failed to get product %s from inventory: %v", itemInput.ProductId, err)
			st, ok := status.FromError(err)
			if ok {
				if st.Code() == codes.NotFound {
					return nil, status.Errorf(codes.FailedPrecondition, "Product not found: %s", itemInput.ProductId)
				}
				if st.Code() == codes.InvalidArgument {
					return nil, status.Errorf(codes.FailedPrecondition, "Invalid Product ID format for inventory check: %s", itemInput.ProductId)
				}
				return nil, status.Errorf(codes.Internal, "Failed to verify product %s: %s", itemInput.ProductId, st.Message())
			}
			return nil, status.Errorf(codes.Internal, "Internal error verifying product %s: %v", itemInput.ProductId, err)
		}

		orderItem := domain.OrderItem{
			ProductID:    itemInput.ProductId,
			Quantity:     int(itemInput.Quantity),
			PriceAtOrder: productInfo.Price,
		}
		orderItems = append(orderItems, orderItem)
		totalAmount += productInfo.Price * float64(itemInput.Quantity)
		log.Printf("Product %s (%s) price %.2f obtained. Requested Qty: %d", productInfo.Name, itemInput.ProductId, productInfo.Price, itemInput.Quantity)
	}

	newOrder := &domain.Order{
		UserID:      req.UserId,
		Items:       orderItems,
		TotalAmount: totalAmount,
		Status:      domain.StatusPending,
	}

	log.Printf("Attempting to create order in DB for user %s with %d items, total: %.2f", req.UserId, len(orderItems), totalAmount)
	createErr := s.orderStore.Create(ctx, newOrder)
	if createErr != nil {
		log.Printf("Error saving order to database: %v", createErr)
		return nil, status.Errorf(codes.Internal, "Failed to create order in database: %v", createErr)
	}

	log.Printf("Order %s created successfully for user %s (stock not checked)", newOrder.ID.Hex(), newOrder.UserID)
	return &pb.OrderResponse{Order: OrderToProto(newOrder)}, nil
}

func (s *OrderServer) GetOrderByID(ctx context.Context, req *pb.GetOrderRequest) (*pb.OrderResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "Order ID is required")
	}
	log.Printf("Received GetOrderByID request for ID: %s", req.Id)

	_, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil {
		log.Printf("Invalid order ID format: %s, error: %v", req.Id, err)
		return nil, status.Errorf(codes.InvalidArgument, "Invalid order ID format: %s", req.Id)
	}

	order, err := s.orderStore.GetByID(ctx, req.Id)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			log.Printf("Order %s not found", req.Id)
			return nil, status.Errorf(codes.NotFound, "Order with ID %s not found", req.Id)
		}
		log.Printf("Failed to get order %s: %v", req.Id, err)
		return nil, status.Errorf(codes.Internal, "Failed to get order: %v", err)
	}

	log.Printf("Order %s found for user %s", req.Id, order.UserID)
	return &pb.OrderResponse{Order: OrderToProto(order)}, nil
}

func (s *OrderServer) UpdateOrderStatus(ctx context.Context, req *pb.UpdateOrderStatusRequest) (*pb.OrderResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "Order ID is required")
	}
	log.Printf("Received UpdateOrderStatus request for ID: %s to status %s", req.Id, req.Status)

	_, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil {
		log.Printf("Invalid order ID format for status update: %s, error: %v", req.Id, err)
		return nil, status.Errorf(codes.InvalidArgument, "Invalid order ID format: %s", req.Id)
	}

	newStatusDomain := OrderStatusFromProto(req.Status)

	isValidDomainStatus := false
	switch newStatusDomain {
	case domain.StatusPending, domain.StatusCompleted, domain.StatusCancelled, domain.StatusFailed:
		isValidDomainStatus = true
	}

	if req.Status == pb.OrderStatus_ORDER_STATUS_UNSPECIFIED || !isValidDomainStatus {
		log.Printf("Invalid target status for order %s: proto status %s (domain status '%s')", req.Id, req.Status, newStatusDomain)
		return nil, status.Errorf(codes.InvalidArgument, "Invalid target status specified: %s", req.Status)
	}

	err = s.orderStore.UpdateStatus(ctx, req.Id, newStatusDomain)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			log.Printf("Order %s not found for status update", req.Id)
			return nil, status.Errorf(codes.NotFound, "Order with ID %s not found to update status", req.Id)
		}
		log.Printf("Failed to update status for order %s: %v", req.Id, err)
		return nil, status.Errorf(codes.Internal, "Failed to update order status: %v", err)
	}

	updatedOrder, err := s.orderStore.GetByID(ctx, req.Id)
	if err != nil {
		log.Printf("Failed to retrieve updated order %s after status change: %v", req.Id, err)
		return nil, status.Errorf(codes.Internal, "Failed to retrieve updated order details after status change: %v", err)
	}

	log.Printf("Order %s status updated successfully to %s (domain: '%s')", req.Id, req.Status, newStatusDomain)
	return &pb.OrderResponse{Order: OrderToProto(updatedOrder)}, nil
}

func (s *OrderServer) ListUserOrders(ctx context.Context, req *pb.ListOrdersRequest) (*pb.ListOrdersResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "User ID is required to list orders")
	}
	log.Printf("Received ListUserOrders request for user %s, PageSize: %d, PageNumber: %d", req.UserId, req.PageSize, req.PageNumber)

	limit := int64(req.PageSize)
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	page := int64(req.PageNumber)
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	log.Printf("Listing orders for user %s with limit %d, offset %d (page %d)", req.UserId, limit, offset, page)
	orders, total, err := s.orderStore.ListByUserID(ctx, req.UserId, limit, offset)
	if err != nil {
		log.Printf("Failed to list orders for user %s: %v", req.UserId, err)
		return nil, status.Errorf(codes.Internal, "Failed to list orders for user %s", req.UserId)
	}

	log.Printf("Found %d orders (total %d) for user %s", len(orders), total, req.UserId)
	return &pb.ListOrdersResponse{
		Orders:     OrdersToProto(orders),
		TotalCount: total,
	}, nil
}
