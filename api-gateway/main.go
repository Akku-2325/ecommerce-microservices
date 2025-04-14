package main

import (
	"awesomeProject2/proxy"
	"log"
	"net/http"
	"os"

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
	inventoryURL := getEnv("INVENTORY_SERVICE_URL", "http://localhost:8081")
	orderURL := getEnv("ORDER_SERVICE_URL", "http://localhost:8082")
	gatewayPort := getEnv("GATEWAY_PORT", "8080")

	router := gin.Default()

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "API Gateway UP"})
	})

	apiV1 := router.Group("/api/v1")

	inventoryProxy := proxy.NewProxyHandler(inventoryURL)
	log.Printf("Proxying /api/v1/products and /api/v1/products/* to %s", inventoryURL)
	apiV1.Any("/products", inventoryProxy)
	apiV1.Any("/products/*proxyPath", inventoryProxy)

	orderProxy := proxy.NewProxyHandler(orderURL)
	log.Printf("Proxying /api/v1/orders and /api/v1/orders/* to %s", orderURL)
	apiV1.Any("/orders", orderProxy)
	apiV1.Any("/orders/*proxyPath", orderProxy)

	serverAddr := ":" + gatewayPort
	log.Printf("Starting API Gateway on %s", serverAddr)

	if err := router.Run(serverAddr); err != nil {
		log.Fatalf("Failed to start API Gateway: %v", err)
	}
}