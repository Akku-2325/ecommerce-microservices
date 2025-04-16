package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	orderpb "ecommerce-microservices/order-service/pb"

	"github.com/gin-gonic/gin"
)

type OrderHandler struct {
	client orderpb.OrderServiceClient
}

func NewOrderHandler(client orderpb.OrderServiceClient) *OrderHandler {
	return &OrderHandler{client: client}
}

func (h *OrderHandler) CreateOrder(c *gin.Context) {
	requestInfo := "CreateOrder"
	var reqBody struct {
		UserID string `json:"user_id" binding:"required"`
		Items  []struct {
			ProductID string `json:"product_id" binding:"required"`
			Quantity  int32  `json:"quantity" binding:"required,gt=0"`
		} `json:"items" binding:"required,min=1,dive"`
	}

	if err := c.ShouldBindJSON(&reqBody); err != nil {
		log.Printf("API Gateway: Invalid input for %s: %v", requestInfo, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	grpcItems := make([]*orderpb.CreateOrderItemInput, len(reqBody.Items))
	for i, item := range reqBody.Items {
		grpcItems[i] = &orderpb.CreateOrderItemInput{
			ProductId: item.ProductID,
			Quantity:  item.Quantity,
		}
	}

	grpcReq := &orderpb.CreateOrderRequest{
		UserId: reqBody.UserID,
		Items:  grpcItems,
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	log.Printf("API Gateway: Calling gRPC %s for user %s with %d items", requestInfo, reqBody.UserID, len(grpcItems))
	resp, err := h.client.CreateOrder(ctx, grpcReq)
	if err != nil {
		mapGrpcToHttpError(c, err, requestInfo)
		return
	}

	log.Printf("API Gateway: gRPC %s successful, order ID: %s", requestInfo, resp.Order.Id)
	c.JSON(http.StatusCreated, resp.Order)
}

func (h *OrderHandler) GetOrderByID(c *gin.Context) {
	orderID := c.Param("id")
	requestInfo := fmt.Sprintf("GetOrderByID (ID: %s)", orderID)

	if orderID == "" {
		log.Printf("API Gateway: Invalid input for %s: Order ID is missing", requestInfo)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order ID is required"})
		return
	}

	grpcReq := &orderpb.GetOrderRequest{Id: orderID}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	log.Printf("API Gateway: Calling gRPC %s", requestInfo)
	resp, err := h.client.GetOrderByID(ctx, grpcReq)
	if err != nil {
		mapGrpcToHttpError(c, err, requestInfo)
		return
	}

	log.Printf("API Gateway: gRPC %s successful", requestInfo)
	c.JSON(http.StatusOK, resp.Order)
}

func (h *OrderHandler) UpdateOrderStatus(c *gin.Context) {
	orderID := c.Param("id")
	requestInfo := fmt.Sprintf("UpdateOrderStatus (ID: %s)", orderID)

	if orderID == "" {
		log.Printf("API Gateway: Invalid input for %s: Order ID is missing", requestInfo)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order ID is required"})
		return
	}

	var reqBody struct {
		Status string `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&reqBody); err != nil {
		log.Printf("API Gateway: Invalid input for %s: %v", requestInfo, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	var grpcStatus orderpb.OrderStatus
	statusStrUpper := strings.ToUpper(reqBody.Status)
	if val, ok := orderpb.OrderStatus_value[statusStrUpper]; ok && orderpb.OrderStatus(val) != orderpb.OrderStatus_ORDER_STATUS_UNSPECIFIED {
		grpcStatus = orderpb.OrderStatus(val)
	} else {
		log.Printf("API Gateway: Invalid status value for %s: '%s'", requestInfo, reqBody.Status)
		validStatuses := []string{}
		for name, val := range orderpb.OrderStatus_value {
			if orderpb.OrderStatus(val) != orderpb.OrderStatus_ORDER_STATUS_UNSPECIFIED {
				validStatuses = append(validStatuses, name) // Собираем валидные имена
			}
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid status value: '%s'. Valid values (case-insensitive): %s", reqBody.Status, strings.Join(validStatuses, ", "))})
		return
	}

	grpcReq := &orderpb.UpdateOrderStatusRequest{
		Id:     orderID,
		Status: grpcStatus,
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	log.Printf("API Gateway: Calling gRPC %s with status %s", requestInfo, grpcStatus)
	resp, err := h.client.UpdateOrderStatus(ctx, grpcReq)
	if err != nil {
		mapGrpcToHttpError(c, err, requestInfo)
		return
	}

	log.Printf("API Gateway: gRPC %s successful", requestInfo)
	c.JSON(http.StatusOK, resp.Order)
}

func (h *OrderHandler) ListUserOrders(c *gin.Context) {
	userID := c.Query("user_id")
	requestInfo := fmt.Sprintf("ListUserOrders (User: %s)", userID)

	if userID == "" {
		log.Printf("API Gateway: Invalid input for %s: Missing 'user_id' query parameter", requestInfo)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing 'user_id' query parameter"})
		return
	}

	pageSizeStr := c.DefaultQuery("page_size", "10")
	pageNumStr := c.DefaultQuery("page", "1")

	pageSize, err1 := strconv.ParseInt(pageSizeStr, 10, 32)
	pageNum, err2 := strconv.ParseInt(pageNumStr, 10, 32)

	if err1 != nil || err2 != nil || pageSize <= 0 || pageNum <= 0 {
		log.Printf("API Gateway: Invalid pagination parameters for %s: page_size=%s, page=%s", requestInfo, pageSizeStr, pageNumStr)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pagination parameters. 'page_size' and 'page' must be positive integers."})
		return
	}
	if pageSize > 100 {
		pageSize = 100
	}

	grpcReq := &orderpb.ListOrdersRequest{
		UserId:     userID,
		PageSize:   int32(pageSize),
		PageNumber: int32(pageNum),
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	log.Printf("API Gateway: Calling gRPC %s with params: %+v", requestInfo, grpcReq)
	resp, err := h.client.ListUserOrders(ctx, grpcReq)
	if err != nil {
		mapGrpcToHttpError(c, err, requestInfo)
		return
	}

	log.Printf("API Gateway: gRPC %s successful, found %d orders (total: %d)", requestInfo, len(resp.Orders), resp.TotalCount)
	c.JSON(http.StatusOK, gin.H{
		"data":      resp.Orders,
		"total":     resp.TotalCount,
		"page":      pageNum,
		"page_size": pageSize,
	})
}
