package main

import (
	"context"
	"ecommerce-microservices/order-service/client"
	"ecommerce-microservices/order-service/handlers"
	"ecommerce-microservices/order-service/store"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/readpref"
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
	mongoCfg := store.MongoConfig{
		Host:     getEnv("MONGO_HOST", "localhost"),
		Port:     getEnv("MONGO_PORT", "27017"),
		User:     getEnv("MONGO_USER", ""),
		Password: getEnv("MONGO_PASSWORD", ""),
		DBName:   getEnv("MONGO_DBNAME", "order_db"),
		Timeout:  15 * time.Second,
	}
	serverPort := getEnv("SERVER_PORT", "8082")
	inventoryServiceURL := getEnv("INVENTORY_SERVICE_URL", "http://localhost:8081")

	var err error
	mongoClient, err = store.NewMongoConnection(mongoCfg)
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

	mongoDB := store.GetMongoDatabase(mongoClient, mongoCfg.DBName)

	orderStore := store.NewMongoOrderStore(mongoDB)

	inventoryClient := client.NewInventoryClient(inventoryServiceURL)

	orderHandler := handlers.NewOrderHandler(orderStore, inventoryClient)

	router := gin.Default()

	router.GET("/health", healthCheckHandler)

	orderRoutes := router.Group("/orders")
	{
		orderRoutes.POST("", orderHandler.CreateOrder)
		orderRoutes.GET("/:id", orderHandler.GetOrderByID)
		orderRoutes.PATCH("/:id", orderHandler.UpdateOrderStatus)
		orderRoutes.GET("", orderHandler.ListOrders)
	}

	serverAddr := ":" + serverPort
	srv := &http.Server{
		Addr:         serverAddr,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Starting Order Service (MongoDB) on %s", serverAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
}

func healthCheckHandler(c *gin.Context) {
	if mongoClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "DOWN", "error": "Mongo client not initialized"})
		return
	}
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer pingCancel()
	err := mongoClient.Ping(pingCtx, readpref.Primary())
	if err != nil {
		log.Printf("Health check failed: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "DOWN", "error": "MongoDB ping failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "UP"})
}
