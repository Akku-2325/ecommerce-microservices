package main

import (
	"context"
	"ecommerce-microservices/order-service/pb"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	invClient "ecommerce-microservices/order-service/internal/client"
	grpcServer "ecommerce-microservices/order-service/internal/delivery/grpc"
	repo "ecommerce-microservices/order-service/internal/repository"

	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	log.Printf("Using fallback for env var %s: %s", key, fallback)
	return fallback
}

var (
	mongoClient            *mongo.Client
	inventoryServiceClient invClient.InventoryClient
	inventoryServiceConn   *grpc.ClientConn
)

func main() {
	mongoCfg := repo.MongoConfig{
		Host:     getEnv("MONGO_HOST", "localhost"),
		Port:     getEnv("MONGO_PORT", "27017"),
		User:     getEnv("MONGO_USER", ""),
		Password: getEnv("MONGO_PASSWORD", ""),
		DBName:   getEnv("MONGO_DBNAME", "order_db"),
		Timeout:  15 * time.Second,
	}
	grpcPort := getEnv("GRPC_PORT", "50052")
	inventoryServiceAddr := getEnv("INVENTORY_SERVICE_ADDR", "localhost:50051")

	var err error

	mongoClient, err = repo.NewMongoConnection(mongoCfg)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer func() {
		log.Println("Disconnecting MongoDB client...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err = mongoClient.Disconnect(ctx); err != nil {
			log.Printf("Error disconnecting MongoDB client: %v", err)
		}
	}()
	mongoDB := repo.GetMongoDatabase(mongoClient, mongoCfg.DBName)

	connCtx, connCancel := context.WithTimeout(context.Background(), 15*time.Second)
	inventoryServiceClient, inventoryServiceConn, err = invClient.NewInventoryGRPCClient(connCtx, inventoryServiceAddr)
	connCancel()
	if err != nil {
		log.Fatalf("Failed to connect to Inventory Service at %s: %v", inventoryServiceAddr, err)
	}
	defer func() {
		log.Println("Closing connection to Inventory Service...")
		if inventoryServiceConn != nil {
			inventoryServiceConn.Close()
		}
	}()

	orderStore := repo.NewMongoOrderStore(mongoDB)
	orderServer := grpcServer.NewOrderServer(orderStore, inventoryServiceClient)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", grpcPort, err)
	}

	srv := grpc.NewServer()
	pb.RegisterOrderServiceServer(srv, orderServer)
	reflection.Register(srv)

	go func() {
		log.Printf("Starting Order gRPC Service on port %s", grpcPort)
		if err := srv.Serve(lis); err != nil {
			if !errors.Is(err, grpc.ErrServerStopped) {
				log.Fatalf("Failed to serve gRPC: %v", err)
			}
		}
		log.Println("gRPC server stopped serving")
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("Received signal %v, shutting down gRPC server...", sig)

	stopped := make(chan struct{})
	go func() {
		srv.GracefulStop()
		close(stopped)
	}()

	t := time.NewTimer(10 * time.Second)
	select {
	case <-t.C:
		log.Println("Graceful shutdown timed out, forcing stop.")
		srv.Stop()
	case <-stopped:
		t.Stop()
		log.Println("gRPC server stopped gracefully.")
	}

	log.Println("Order Service exiting")
}
