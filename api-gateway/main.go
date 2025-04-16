package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ecommerce-microservices/api-gateway/internal/client"
	"ecommerce-microservices/api-gateway/internal/handlers"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	log.Printf("Using fallback for env var %s: %s", key, fallback)
	return fallback
}

func main() {
	// Адреса gRPC сервисов из переменных окружения
	inventoryAddr := getEnv("INVENTORY_SERVICE_ADDR", "localhost:50051")
	orderAddr := getEnv("ORDER_SERVICE_ADDR", "localhost:50052")
	gatewayPort := getEnv("GATEWAY_PORT", "8080")

	log.Println("API Gateway: Initializing gRPC clients...")
	serviceClients, err := client.NewServiceClients(inventoryAddr, orderAddr)
	if err != nil {
		// Это фатально, Gateway не может работать без бэкендов
		log.Fatalf("API Gateway: Failed to create gRPC service clients: %v", err)
	}
	defer serviceClients.Close()
	log.Println("API Gateway: gRPC clients initialized.")

	ginMode := getEnv("GIN_MODE", "release")
	gin.SetMode(ginMode)

	router := gin.New()

	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	invHandler := handlers.NewInventoryHandler(serviceClients.Inventory)
	ordHandler := handlers.NewOrderHandler(serviceClients.Order)

	router.GET("/health", func(c *gin.Context) {
		// TODO: Можно добавить пинги gRPC сервисов для более полной проверки, если нужно
		c.JSON(http.StatusOK, gin.H{"status": "API Gateway UP"})
	})

	// Группа роутов /api/v1
	apiV1 := router.Group("/api/v1")
	{
		products := apiV1.Group("/products")
		{
			log.Printf("API Gateway: Registering route POST /api/v1/products")
			products.POST("", invHandler.CreateProduct) // POST /api/v1/products

			log.Printf("API Gateway: Registering route GET /api/v1/products/:id")
			products.GET("/:id", invHandler.GetProductByID) // GET /api/v1/products/{product_id}

			log.Printf("API Gateway: Registering route PUT /api/v1/products/:id")
			products.PUT("/:id", invHandler.UpdateProduct) // PUT /api/v1/products/{product_id}

			log.Printf("API Gateway: Registering route DELETE /api/v1/products/:id")
			products.DELETE("/:id", invHandler.DeleteProduct) // DELETE /api/v1/products/{product_id}

			log.Printf("API Gateway: Registering route GET /api/v1/products")
			products.GET("", invHandler.ListProducts) // GET /api/v1/products
		}

		//РОУТЫ ДЛЯ КАТЕГОРИЙ
		categories := apiV1.Group("/categories")
		{
			log.Printf("API Gateway: Registering route POST /api/v1/categories")
			categories.POST("", invHandler.CreateCategory) // POST /api/v1/categories

			log.Printf("API Gateway: Registering route GET /api/v1/categories/:id")
			categories.GET("/:id", invHandler.GetCategoryByID) // GET /api/v1/categories/{category_id}

			log.Printf("API Gateway: Registering route PUT /api/v1/categories/:id")
			categories.PUT("/:id", invHandler.UpdateCategory) // PUT /api/v1/categories/{category_id}

			log.Printf("API Gateway: Registering route DELETE /api/v1/categories/:id")
			categories.DELETE("/:id", invHandler.DeleteCategory) // DELETE /api/v1/categories/{category_id}

			log.Printf("API Gateway: Registering route GET /api/v1/categories")
			categories.GET("", invHandler.ListCategories) // GET /api/v1/categories
		}

		// Роуты для Order
		orders := apiV1.Group("/orders")
		{
			log.Printf("API Gateway: Registering route POST /api/v1/orders")
			orders.POST("", ordHandler.CreateOrder) // POST /api/v1/orders

			log.Printf("API Gateway: Registering route GET /api/v1/orders/:id")
			orders.GET("/:id", ordHandler.GetOrderByID) // GET /api/v1/orders/{order_id}

			log.Printf("API Gateway: Registering route PATCH /api/v1/orders/:id")
			orders.PATCH("/:id", ordHandler.UpdateOrderStatus) // PATCH /api/v1/orders/{order_id} (для статуса)

			log.Printf("API Gateway: Registering route GET /api/v1/orders")
			orders.GET("", ordHandler.ListUserOrders) // GET /api/v1/orders?user_id=...
		}
	}

	serverAddr := ":" + gatewayPort
	srv := &http.Server{
		Addr:         serverAddr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("API Gateway: Starting HTTP server on %s", serverAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("API Gateway: ListenAndServe error: %s\n", err)
		}
		log.Println("API Gateway: HTTP server stopped serving.")
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("API Gateway: Received signal %v, shutting down server...", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("API Gateway: Server forced to shutdown: %v", err)
	}

	log.Println("API Gateway: Server exiting")
}
