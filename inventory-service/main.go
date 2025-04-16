package main

import (
	"context"
	grpcServer "ecommerce-microservices/inventory-service/internal/delivery/grpc"
	repo "ecommerce-microservices/inventory-service/internal/repository"
	pb "ecommerce-microservices/inventory-service/pb"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	mongoClient *mongo.Client
)

func main() {
	mongoCfg := repo.MongoConfig{
		Host:     getEnv("MONGO_HOST", "localhost"),
		Port:     getEnv("MONGO_PORT", "27017"),
		User:     getEnv("MONGO_USER", ""),
		Password: getEnv("MONGO_PASSWORD", ""),
		DBName:   getEnv("MONGO_DBNAME", "inventory_db"),
		Timeout:  15 * time.Second,
	}
	grpcPort := getEnv("GRPC_PORT", "50051")

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

	productStore := repo.NewMongoProductStore(mongoDB)
	categoryStore := repo.NewMongoCategoryStore(mongoDB)

	inventoryGrpcServer := grpcServer.NewInventoryServer(productStore, categoryStore)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", grpcPort, err)
	}

	grpcServer := grpc.NewServer()

	pb.RegisterInventoryServiceServer(grpcServer, inventoryGrpcServer)

	reflection.Register(grpcServer)

	go func() {
		log.Printf("Starting Inventory gRPC Service on port %s", grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down gRPC server...")

	grpcServer.GracefulStop()

	log.Println("Server exiting")
}
