package client

import (
	"context"
	inventorypb "ecommerce-microservices/inventory-service/pb"
	"fmt"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type InventoryClient interface {
	GetProduct(ctx context.Context, productID string) (*inventorypb.Product, error)
}

type grpcInventoryClient struct {
	conn inventorypb.InventoryServiceClient
}

func NewInventoryGRPCClient(ctx context.Context, target string) (InventoryClient, *grpc.ClientConn, error) {
	log.Printf("Connecting to Inventory gRPC Service at %s", target)
	conn, err := grpc.DialContext(ctx, target, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to dial inventory service: %w", err)
	}

	client := inventorypb.NewInventoryServiceClient(conn)
	log.Printf("Successfully connected to Inventory gRPC Service")

	return &grpcInventoryClient{conn: client}, conn, nil
}

func (c *grpcInventoryClient) GetProduct(ctx context.Context, productID string) (*inventorypb.Product, error) {
	log.Printf("gRPC Client: Calling InventoryService.GetProductByID for ID: %s", productID)
	req := &inventorypb.GetProductRequest{Id: productID}

	resp, err := c.conn.GetProductByID(ctx, req)
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			log.Printf("gRPC Client: InventoryService.GetProductByID failed with code %s: %s", st.Code(), st.Message())
			return nil, err
		}
		log.Printf("gRPC Client: InventoryService.GetProductByID failed with non-gRPC error: %v", err)
		return nil, fmt.Errorf("inventory service call failed: %w", err)
	}
	if resp == nil || resp.Product == nil {
		log.Printf("gRPC Client: Received nil product response for ID: %s", productID)
		return nil, status.Error(codes.Internal, "received nil product from inventory service")
	}

	log.Printf("gRPC Client: Received product details for ID: %s, Name: %s", productID, resp.Product.Name)
	return resp.Product, nil
}
