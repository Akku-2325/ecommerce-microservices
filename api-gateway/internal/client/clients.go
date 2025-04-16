package client

import (
	"context"
	inventorypb "ecommerce-microservices/inventory-service/pb"
	orderpb "ecommerce-microservices/order-service/pb"
	"fmt"
	"log"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ServiceClients struct {
	Inventory inventorypb.InventoryServiceClient
	Order     orderpb.OrderServiceClient
	invConn   *grpc.ClientConn
	ordConn   *grpc.ClientConn
}

func NewServiceClients(invTarget, ordTarget string) (*ServiceClients, error) {
	var wg sync.WaitGroup
	var invClient inventorypb.InventoryServiceClient
	var ordClient orderpb.OrderServiceClient
	var invConn *grpc.ClientConn
	var ordConn *grpc.ClientConn
	var invErr, ordErr error

	connTimeout := 15 * time.Second

	wg.Add(2)

	// Подключение к Inventory
	go func() {
		defer wg.Done()
		log.Printf("API Gateway: Connecting to Inventory gRPC Service at %s", invTarget)
		ctx, cancel := context.WithTimeout(context.Background(), connTimeout)
		defer cancel()
		conn, err := grpc.DialContext(ctx, invTarget,
			grpc.WithTransportCredentials(insecure.NewCredentials()), // Используем insecure для простоты
			grpc.WithBlock(), // Ждем установления соединения
		)
		if err != nil {
			invErr = fmt.Errorf("failed to dial inventory service (%s): %w", invTarget, err)
			log.Printf("ERROR: %v", invErr)
			return
		}
		invConn = conn
		invClient = inventorypb.NewInventoryServiceClient(invConn)
		log.Printf("API Gateway: Successfully connected to Inventory gRPC Service")
	}()

	// Подключение к Order
	go func() {
		defer wg.Done()
		log.Printf("API Gateway: Connecting to Order gRPC Service at %s", ordTarget)
		ctx, cancel := context.WithTimeout(context.Background(), connTimeout)
		defer cancel()
		conn, err := grpc.DialContext(ctx, ordTarget,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
		)
		if err != nil {
			ordErr = fmt.Errorf("failed to dial order service (%s): %w", ordTarget, err)
			log.Printf("ERROR: %v", ordErr)
			return
		}
		ordConn = conn
		ordClient = orderpb.NewOrderServiceClient(ordConn)
		log.Printf("API Gateway: Successfully connected to Order gRPC Service")
	}()

	wg.Wait()

	if invErr != nil {
		if ordConn != nil {
			ordConn.Close()
		}
		return nil, invErr
	}
	if ordErr != nil {
		if invConn != nil {
			invConn.Close()
		}
		return nil, ordErr
	}

	return &ServiceClients{
		Inventory: invClient,
		Order:     ordClient,
		invConn:   invConn,
		ordConn:   ordConn,
	}, nil
}

func (c *ServiceClients) Close() {
	log.Println("API Gateway: Closing gRPC client connections...")
	if c.invConn != nil {
		err := c.invConn.Close()
		if err != nil {
			log.Printf("API Gateway: Error closing inventory connection: %v", err)
		}
	}
	if c.ordConn != nil {
		err := c.ordConn.Close()
		if err != nil {
			log.Printf("API Gateway: Error closing order connection: %v", err)
		}
	}
}
