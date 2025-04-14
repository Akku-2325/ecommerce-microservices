package handlers

import (
	"ecommerce-microservices/order-service/client"
	"ecommerce-microservices/order-service/models"
	"ecommerce-microservices/order-service/store"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type OrderHandler struct {
	store           *store.MongoOrderStore
	inventoryClient *client.InventoryClient
}

func NewOrderHandler(s *store.MongoOrderStore, invClient *client.InventoryClient) *OrderHandler {
	return &OrderHandler{store: s, inventoryClient: invClient}
}

func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var input models.CreateOrderInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	var orderItems []models.OrderItem
	var totalAmount float64
	productIDs := make(map[string]bool)

	for _, itemInput := range input.Items {
		// Check for duplicates
		if _, exists := productIDs[itemInput.ProductID]; exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Duplicate product ID in order: %s", itemInput.ProductID)})
			return
		}
		productIDs[itemInput.ProductID] = true

		log.Printf("Fetching details for product ID: %s", itemInput.ProductID)
		productInfo, err := h.inventoryClient.GetProductDetails(c.Request.Context(), itemInput.ProductID)
		if err != nil {
			log.Printf("Error fetching product %s: %v", itemInput.ProductID, err)
			errorMsg := fmt.Sprintf("Failed to process product %s: %s", itemInput.ProductID, err.Error())
			statusCode := http.StatusInternalServerError
			if errors.Is(err, client.ErrProductNotFound) {
				statusCode = http.StatusBadRequest
				errorMsg = fmt.Sprintf("Product not found: %s", itemInput.ProductID)
			} else if errors.Is(err, client.ErrInventoryServiceUnavailable) {
				statusCode = http.StatusBadGateway
			}
			c.JSON(statusCode, gin.H{"error": errorMsg})
			return
		}

		if productInfo.Stock < itemInput.Quantity {
			log.Printf("Insufficient stock for product %s: requested %d, available %d", itemInput.ProductID, itemInput.Quantity, productInfo.Stock)
			errorMsg := fmt.Sprintf("Insufficient stock for product %s (requested %d, available %d)", productInfo.Name, itemInput.Quantity, productInfo.Stock)
			c.JSON(http.StatusConflict, gin.H{"error": errorMsg})
			return
		}

		orderItem := models.OrderItem{
			ProductID:    itemInput.ProductID,
			Quantity:     itemInput.Quantity,
			PriceAtOrder: productInfo.Price,
		}
		orderItems = append(orderItems, orderItem)
		totalAmount += productInfo.Price * float64(itemInput.Quantity)
	}

	newOrder := models.Order{
		UserID:      input.UserID,
		Items:       orderItems,
		TotalAmount: totalAmount,
		Status:      models.StatusPending,
	}

	err := h.store.Create(c.Request.Context(), &newOrder)
	if err != nil {
		log.Printf("Error saving order: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order: " + err.Error()})
		return
	}

	log.Printf("Order %s created successfully for user %s", newOrder.ID.Hex(), newOrder.UserID)
	c.JSON(http.StatusCreated, newOrder)
}

func (h *OrderHandler) GetOrderByID(c *gin.Context) {
	id := c.Param("id")
	order, err := h.store.GetByID(c.Request.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "invalid id format") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve order: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, order)
}

func (h *OrderHandler) UpdateOrderStatus(c *gin.Context) {
	id := c.Param("id")
	var input models.UpdateOrderStatusInput

	_, err := h.store.GetByID(c.Request.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "invalid id format") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve order for update: " + err.Error()})
		}
		return
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	validStatuses := map[models.OrderStatus]bool{
		models.StatusPending:   true,
		models.StatusCompleted: true,
		models.StatusCancelled: true,
		models.StatusFailed:    true,
	}
	if !validStatuses[input.Status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid target status: " + string(input.Status)})
		return
	}

	err = h.store.UpdateStatus(c.Request.Context(), id, input.Status)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Order not found to update status"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update order status: " + err.Error()})
		}
		return
	}

	updatedOrder, err := h.store.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve updated order: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, updatedOrder)
}

func (h *OrderHandler) ListOrders(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing user_id query parameter"})
		return
	}

	limitStr := c.DefaultQuery("limit", "10")
	offsetStr := c.DefaultQuery("offset", "0")
	limit, errL := strconv.ParseInt(limitStr, 10, 64)
	offset, errO := strconv.ParseInt(offsetStr, 10, 64)
	if errL != nil || errO != nil || limit <= 0 || offset < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pagination parameters"})
		return
	}
	if limit > 100 {
		limit = 100
	}

	orders, total, err := h.store.ListByUserID(c.Request.Context(), userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list orders: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": orders,
		"pagination": gin.H{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}
