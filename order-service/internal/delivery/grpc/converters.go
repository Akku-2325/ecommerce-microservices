package grpc

import (
	"ecommerce-microservices/order-service/internal/domain"
	"ecommerce-microservices/order-service/pb"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// --- Order Converters ---

func OrderStatusToProto(status domain.OrderStatus) pb.OrderStatus {
	switch status {
	case domain.StatusPending:
		return pb.OrderStatus_PENDING
	case domain.StatusCompleted:
		return pb.OrderStatus_COMPLETED
	case domain.StatusCancelled:
		return pb.OrderStatus_CANCELLED
	case domain.StatusFailed:
		return pb.OrderStatus_FAILED
	default:
		return pb.OrderStatus_ORDER_STATUS_UNSPECIFIED
	}
}

func OrderStatusFromProto(status pb.OrderStatus) domain.OrderStatus {
	switch status {
	case pb.OrderStatus_PENDING:
		return domain.StatusPending
	case pb.OrderStatus_COMPLETED:
		return domain.StatusCompleted
	case pb.OrderStatus_CANCELLED:
		return domain.StatusCancelled
	case pb.OrderStatus_FAILED:
		return domain.StatusFailed
	default:
		return domain.StatusPending // Возвращаем Pending как статус по умолчанию при ошибке
	}
}

func OrderItemToProto(item domain.OrderItem) *pb.OrderItem {
	return &pb.OrderItem{
		ProductId:    item.ProductID,
		Quantity:     int32(item.Quantity),
		PriceAtOrder: item.PriceAtOrder,
	}
}

func OrderItemsToProto(items []domain.OrderItem) []*pb.OrderItem {
	if items == nil {
		return []*pb.OrderItem{}
	}
	protoItems := make([]*pb.OrderItem, len(items))
	for i, item := range items {
		protoItems[i] = OrderItemToProto(item)
	}
	return protoItems
}

func OrderToProto(o *domain.Order) *pb.Order {
	if o == nil {
		return nil
	}
	return &pb.Order{
		Id:          o.ID.Hex(),
		UserId:      o.UserID,
		Items:       OrderItemsToProto(o.Items),
		TotalAmount: o.TotalAmount,
		Status:      OrderStatusToProto(o.Status),
		CreatedAt:   timestamppb.New(o.CreatedAt),
		UpdatedAt:   timestamppb.New(o.UpdatedAt),
	}
}

func OrdersToProto(orders []*domain.Order) []*pb.Order {
	if orders == nil {
		return []*pb.Order{}
	}
	protoOrders := make([]*pb.Order, len(orders))
	for i, o := range orders {
		protoOrders[i] = OrderToProto(o)
	}
	return protoOrders
}
