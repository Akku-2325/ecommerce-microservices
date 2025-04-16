package grpc

import (
	"context"
	"ecommerce-microservices/inventory-service/internal/domain"
	repo "ecommerce-microservices/inventory-service/internal/repository"
	pb "ecommerce-microservices/inventory-service/pb"
	"errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"strings"
	// "log"
)

type InventoryServer struct {
	pb.UnimplementedInventoryServiceServer
	productStore  *repo.MongoProductStore
	categoryStore *repo.MongoCategoryStore
}

func NewInventoryServer(ps *repo.MongoProductStore, cs *repo.MongoCategoryStore) *InventoryServer {
	return &InventoryServer{
		productStore:  ps,
		categoryStore: cs,
	}
}

func (s *InventoryServer) CreateProduct(ctx context.Context, req *pb.CreateProductRequest) (*pb.ProductResponse, error) {
	if req.Name == "" || req.Price <= 0 || req.CategoryId == "" {
		return nil, status.Error(codes.InvalidArgument, "Name, positive price, and category ID are required")
	}

	product := &domain.Product{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Stock:       int(req.Stock),
		CategoryID:  req.CategoryId,
	}

	err := s.productStore.Create(ctx, product)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to create product: %v", err)
	}

	return &pb.ProductResponse{Product: ProductToProto(product)}, nil
}

func (s *InventoryServer) GetProductByID(ctx context.Context, req *pb.GetProductRequest) (*pb.ProductResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "Product ID is required")
	}

	product, err := s.productStore.GetByID(ctx, req.Id)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) || strings.Contains(err.Error(), "not found") {
			return nil, status.Errorf(codes.NotFound, "Product with ID %s not found", req.Id)
		}
		if strings.Contains(err.Error(), "invalid id format") {
			return nil, status.Errorf(codes.InvalidArgument, "Invalid product ID format: %s", req.Id)
		}
		return nil, status.Errorf(codes.Internal, "Failed to get product: %v", err)
	}

	return &pb.ProductResponse{Product: ProductToProto(product)}, nil
}

func (s *InventoryServer) UpdateProduct(ctx context.Context, req *pb.UpdateProductRequest) (*pb.ProductResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "Product ID is required for update")
	}
	if req.Name == "" || req.Price <= 0 || req.CategoryId == "" {
		return nil, status.Error(codes.InvalidArgument, "Name, positive price, and category ID are required")
	}

	product := &domain.Product{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Stock:       int(req.Stock),
		CategoryID:  req.CategoryId,
	}

	err := s.productStore.Update(ctx, req.Id, product)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Errorf(codes.NotFound, "Product with ID %s not found for update", req.Id)
		}
		if strings.Contains(err.Error(), "invalid id format") {
			return nil, status.Errorf(codes.InvalidArgument, "Invalid product ID format: %s", req.Id)
		}
		return nil, status.Errorf(codes.Internal, "Failed to update product: %v", err)
	}

	updatedProduct, err := s.productStore.GetByID(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to retrieve updated product: %v", err)
	}

	return &pb.ProductResponse{Product: ProductToProto(updatedProduct)}, nil
}

func (s *InventoryServer) DeleteProduct(ctx context.Context, req *pb.DeleteProductRequest) (*emptypb.Empty, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "Product ID is required")
	}

	err := s.productStore.Delete(ctx, req.Id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Errorf(codes.NotFound, "Product with ID %s not found for deletion", req.Id)
		}
		if strings.Contains(err.Error(), "invalid id format") {
			return nil, status.Errorf(codes.InvalidArgument, "Invalid product ID format: %s", req.Id)
		}
		return nil, status.Errorf(codes.Internal, "Failed to delete product: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (s *InventoryServer) ListProducts(ctx context.Context, req *pb.ListProductsRequest) (*pb.ListProductsResponse, error) {
	limit := int64(req.PageSize)
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	offset := int64(req.PageNumber-1) * limit
	if offset < 0 {
		offset = 0
	}

	filter := bson.M{}
	if req.CategoryIdFilter != "" {
		filter["category_id"] = req.CategoryIdFilter
	}

	products, total, err := s.productStore.List(ctx, filter, limit, offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to list products: %v", err)
	}

	return &pb.ListProductsResponse{
		Products:   ProductsToProto(products),
		TotalCount: total,
	}, nil
}

func (s *InventoryServer) CreateCategory(ctx context.Context, req *pb.CreateCategoryRequest) (*pb.CategoryResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "Category name is required")
	}
	category := &domain.Category{Name: req.Name}
	err := s.categoryStore.Create(ctx, category)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return nil, status.Errorf(codes.AlreadyExists, "Category '%s' already exists", req.Name)
		}
		return nil, status.Errorf(codes.Internal, "Failed to create category: %v", err)
	}
	return &pb.CategoryResponse{Category: CategoryToProto(category)}, nil
}

func (s *InventoryServer) GetCategoryByID(ctx context.Context, req *pb.GetCategoryRequest) (*pb.CategoryResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "Category ID is required")
	}
	category, err := s.categoryStore.GetByID(ctx, req.Id)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) || strings.Contains(err.Error(), "not found") {
			return nil, status.Errorf(codes.NotFound, "Category with ID %s not found", req.Id)
		}
		if strings.Contains(err.Error(), "invalid id format") {
			return nil, status.Errorf(codes.InvalidArgument, "Invalid category ID format: %s", req.Id)
		}
		return nil, status.Errorf(codes.Internal, "Failed to get category: %v", err)
	}
	return &pb.CategoryResponse{Category: CategoryToProto(category)}, nil
}

func (s *InventoryServer) UpdateCategory(ctx context.Context, req *pb.UpdateCategoryRequest) (*pb.CategoryResponse, error) {
	if req.Id == "" || req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "Category ID and name are required for update")
	}
	category := &domain.Category{Name: req.Name}
	err := s.categoryStore.Update(ctx, req.Id, category)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Errorf(codes.NotFound, "Category with ID %s not found for update", req.Id)
		}
		if strings.Contains(err.Error(), "invalid id format") {
			return nil, status.Errorf(codes.InvalidArgument, "Invalid category ID format: %s", req.Id)
		}
		if strings.Contains(err.Error(), "already exists") {
			return nil, status.Errorf(codes.AlreadyExists, "Category name '%s' already exists", req.Name)
		}
		return nil, status.Errorf(codes.Internal, "Failed to update category: %v", err)
	}
	updatedCategory, err := s.categoryStore.GetByID(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to retrieve updated category: %v", err)
	}
	return &pb.CategoryResponse{Category: CategoryToProto(updatedCategory)}, nil
}

func (s *InventoryServer) DeleteCategory(ctx context.Context, req *pb.DeleteCategoryRequest) (*emptypb.Empty, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "Category ID is required")
	}
	err := s.categoryStore.Delete(ctx, req.Id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Errorf(codes.NotFound, "Category with ID %s not found for deletion", req.Id)
		}
		if strings.Contains(err.Error(), "invalid id format") {
			return nil, status.Errorf(codes.InvalidArgument, "Invalid category ID format: %s", req.Id)
		}
		return nil, status.Errorf(codes.Internal, "Failed to delete category: %v", err)
	}
	return &emptypb.Empty{}, nil
}

func (s *InventoryServer) ListCategories(ctx context.Context, req *pb.ListCategoriesRequest) (*pb.ListCategoriesResponse, error) {
	categories, err := s.categoryStore.List(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to list categories: %v", err)
	}
	return &pb.ListCategoriesResponse{Categories: CategoriesToProto(categories)}, nil
}
