package grpc

import (
	"ecommerce-microservices/inventory-service/internal/domain"
	"ecommerce-microservices/inventory-service/pb"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func ProductToProto(p *domain.Product) *pb.Product {
	if p == nil {
		return nil
	}
	return &pb.Product{
		Id:          p.ID.Hex(),
		Name:        p.Name,
		Description: p.Description,
		Price:       p.Price,
		Stock:       int32(p.Stock),
		CategoryId:  p.CategoryID,
		CreatedAt:   timestamppb.New(p.CreatedAt),
		UpdatedAt:   timestamppb.New(p.UpdatedAt),
	}
}

func ProductsToProto(products []*domain.Product) []*pb.Product {
	if products == nil {
		return nil
	}
	protoProducts := make([]*pb.Product, len(products))
	for i, p := range products {
		protoProducts[i] = ProductToProto(p)
	}
	return protoProducts
}

func CategoryToProto(cat *domain.Category) *pb.Category {
	if cat == nil {
		return nil
	}
	return &pb.Category{
		Id:        cat.ID.Hex(),
		Name:      cat.Name,
		CreatedAt: timestamppb.New(cat.CreatedAt),
		UpdatedAt: timestamppb.New(cat.UpdatedAt),
	}
}

func CategoriesToProto(cats []*domain.Category) []*pb.Category {
	if cats == nil {
		return nil
	}
	protoCats := make([]*pb.Category, len(cats))
	for i, c := range cats {
		protoCats[i] = CategoryToProto(c)
	}
	return protoCats
}
