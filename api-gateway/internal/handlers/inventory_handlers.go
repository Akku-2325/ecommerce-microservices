package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	inventorypb "ecommerce-microservices/inventory-service/pb"

	"github.com/gin-gonic/gin"
)

type InventoryHandler struct {
	client inventorypb.InventoryServiceClient
}

func NewInventoryHandler(client inventorypb.InventoryServiceClient) *InventoryHandler {
	return &InventoryHandler{client: client}
}

func (h *InventoryHandler) CreateProduct(c *gin.Context) {
	requestInfo := "CreateProduct"
	var reqBody struct {
		Name        string  `json:"name" binding:"required"`
		Description string  `json:"description"`
		Price       float64 `json:"price" binding:"required,gt=0"`
		Stock       int32   `json:"stock" binding:"gte=0"`
		CategoryID  string  `json:"category_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&reqBody); err != nil {
		log.Printf("API Gateway: Invalid input for %s: %v", requestInfo, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	grpcReq := &inventorypb.CreateProductRequest{
		Name:        reqBody.Name,
		Description: reqBody.Description,
		Price:       reqBody.Price,
		Stock:       reqBody.Stock,
		CategoryId:  reqBody.CategoryID,
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	log.Printf("API Gateway: Calling gRPC %s with data: %+v", requestInfo, grpcReq)
	resp, err := h.client.CreateProduct(ctx, grpcReq)
	if err != nil {
		mapGrpcToHttpError(c, err, requestInfo)
		return
	}

	log.Printf("API Gateway: gRPC %s successful, product ID: %s", requestInfo, resp.Product.Id)
	c.JSON(http.StatusCreated, resp.Product)
}

func (h *InventoryHandler) GetProductByID(c *gin.Context) {
	productID := c.Param("id")
	requestInfo := fmt.Sprintf("GetProductByID (ID: %s)", productID)

	if productID == "" {
		log.Printf("API Gateway: Invalid input for %s: Product ID is missing", requestInfo)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Product ID is required"})
		return
	}

	grpcReq := &inventorypb.GetProductRequest{Id: productID}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	log.Printf("API Gateway: Calling gRPC %s", requestInfo)
	resp, err := h.client.GetProductByID(ctx, grpcReq)
	if err != nil {
		mapGrpcToHttpError(c, err, requestInfo)
		return
	}

	log.Printf("API Gateway: gRPC %s successful", requestInfo)
	c.JSON(http.StatusOK, resp.Product)
}

func (h *InventoryHandler) UpdateProduct(c *gin.Context) {
	productID := c.Param("id")
	requestInfo := fmt.Sprintf("UpdateProduct (ID: %s)", productID)

	if productID == "" {
		log.Printf("API Gateway: Invalid input for %s: Product ID is missing", requestInfo)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Product ID is required"})
		return
	}

	var reqBody struct {
		Name        string  `json:"name" binding:"required"`
		Description string  `json:"description"`
		Price       float64 `json:"price" binding:"required,gt=0"`
		Stock       int32   `json:"stock" binding:"gte=0"`
		CategoryID  string  `json:"category_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&reqBody); err != nil {
		log.Printf("API Gateway: Invalid input for %s: %v", requestInfo, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	grpcReq := &inventorypb.UpdateProductRequest{
		Id:          productID, // ID из URL
		Name:        reqBody.Name,
		Description: reqBody.Description,
		Price:       reqBody.Price,
		Stock:       reqBody.Stock,
		CategoryId:  reqBody.CategoryID,
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	log.Printf("API Gateway: Calling gRPC %s with data: %+v", requestInfo, grpcReq)
	resp, err := h.client.UpdateProduct(ctx, grpcReq)
	if err != nil {
		mapGrpcToHttpError(c, err, requestInfo)
		return
	}

	log.Printf("API Gateway: gRPC %s successful", requestInfo)
	c.JSON(http.StatusOK, resp.Product)
}

func (h *InventoryHandler) DeleteProduct(c *gin.Context) {
	productID := c.Param("id")
	requestInfo := fmt.Sprintf("DeleteProduct (ID: %s)", productID)

	if productID == "" {
		log.Printf("API Gateway: Invalid input for %s: Product ID is missing", requestInfo)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Product ID is required"})
		return
	}

	grpcReq := &inventorypb.DeleteProductRequest{Id: productID}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	log.Printf("API Gateway: Calling gRPC %s", requestInfo)
	_, err := h.client.DeleteProduct(ctx, grpcReq) // Ответ пустой (Empty)
	if err != nil {
		mapGrpcToHttpError(c, err, requestInfo)
		return
	}

	log.Printf("API Gateway: gRPC %s successful", requestInfo)
	c.Status(http.StatusNoContent)
}

func (h *InventoryHandler) ListProducts(c *gin.Context) {
	requestInfo := "ListProducts"
	pageSizeStr := c.DefaultQuery("page_size", "10")
	pageNumStr := c.DefaultQuery("page", "1")
	categoryFilter := c.Query("category_id")

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

	grpcReq := &inventorypb.ListProductsRequest{
		PageSize:         int32(pageSize),
		PageNumber:       int32(pageNum),
		CategoryIdFilter: categoryFilter,
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	log.Printf("API Gateway: Calling gRPC %s with params: %+v", requestInfo, grpcReq)
	resp, err := h.client.ListProducts(ctx, grpcReq)
	if err != nil {
		mapGrpcToHttpError(c, err, requestInfo)
		return
	}

	log.Printf("API Gateway: gRPC %s successful, found %d products (total: %d)", requestInfo, len(resp.Products), resp.TotalCount)
	c.JSON(http.StatusOK, gin.H{
		"data":            resp.Products,
		"total":           resp.TotalCount,
		"page":            pageNum,
		"page_size":       pageSize,
		"category_filter": categoryFilter,
	})
}

//Хендлеры для Категорий

func (h *InventoryHandler) CreateCategory(c *gin.Context) {
	requestInfo := "CreateCategory"
	var reqBody struct {
		Name string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&reqBody); err != nil {
		log.Printf("API Gateway: Invalid input for %s: %v", requestInfo, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	grpcReq := &inventorypb.CreateCategoryRequest{Name: reqBody.Name}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	log.Printf("API Gateway: Calling gRPC %s with name: %s", requestInfo, grpcReq.Name)
	resp, err := h.client.CreateCategory(ctx, grpcReq)
	if err != nil {
		mapGrpcToHttpError(c, err, requestInfo)
		return
	}

	log.Printf("API Gateway: gRPC %s successful, category ID: %s", requestInfo, resp.Category.Id)
	c.JSON(http.StatusCreated, resp.Category)
}

func (h *InventoryHandler) GetCategoryByID(c *gin.Context) {
	categoryID := c.Param("id")
	requestInfo := fmt.Sprintf("GetCategoryByID (ID: %s)", categoryID)

	if categoryID == "" {
		log.Printf("API Gateway: Invalid input for %s: Category ID is missing", requestInfo)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Category ID is required"})
		return
	}

	grpcReq := &inventorypb.GetCategoryRequest{Id: categoryID}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	log.Printf("API Gateway: Calling gRPC %s", requestInfo)
	resp, err := h.client.GetCategoryByID(ctx, grpcReq)
	if err != nil {
		mapGrpcToHttpError(c, err, requestInfo)
		return
	}

	log.Printf("API Gateway: gRPC %s successful", requestInfo)
	c.JSON(http.StatusOK, resp.Category)
}

func (h *InventoryHandler) UpdateCategory(c *gin.Context) {
	categoryID := c.Param("id")
	requestInfo := fmt.Sprintf("UpdateCategory (ID: %s)", categoryID)

	if categoryID == "" {
		log.Printf("API Gateway: Invalid input for %s: Category ID is missing", requestInfo)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Category ID is required"})
		return
	}

	var reqBody struct {
		Name string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&reqBody); err != nil {
		log.Printf("API Gateway: Invalid input for %s: %v", requestInfo, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	grpcReq := &inventorypb.UpdateCategoryRequest{
		Id:   categoryID,
		Name: reqBody.Name,
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	log.Printf("API Gateway: Calling gRPC %s with data: %+v", requestInfo, grpcReq)
	resp, err := h.client.UpdateCategory(ctx, grpcReq)
	if err != nil {
		mapGrpcToHttpError(c, err, requestInfo)
		return
	}

	log.Printf("API Gateway: gRPC %s successful", requestInfo)
	c.JSON(http.StatusOK, resp.Category)
}

func (h *InventoryHandler) DeleteCategory(c *gin.Context) {
	categoryID := c.Param("id")
	requestInfo := fmt.Sprintf("DeleteCategory (ID: %s)", categoryID)

	if categoryID == "" {
		log.Printf("API Gateway: Invalid input for %s: Category ID is missing", requestInfo)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Category ID is required"})
		return
	}

	grpcReq := &inventorypb.DeleteCategoryRequest{Id: categoryID}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	log.Printf("API Gateway: Calling gRPC %s", requestInfo)
	_, err := h.client.DeleteCategory(ctx, grpcReq) // Ответ пустой (Empty)
	if err != nil {
		mapGrpcToHttpError(c, err, requestInfo)
		return
	}

	log.Printf("API Gateway: gRPC %s successful", requestInfo)
	c.Status(http.StatusNoContent)
}

func (h *InventoryHandler) ListCategories(c *gin.Context) {
	requestInfo := "ListCategories"

	grpcReq := &inventorypb.ListCategoriesRequest{}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	log.Printf("API Gateway: Calling gRPC %s", requestInfo)
	resp, err := h.client.ListCategories(ctx, grpcReq)
	if err != nil {
		mapGrpcToHttpError(c, err, requestInfo)
		return
	}

	log.Printf("API Gateway: gRPC %s successful, found %d categories", requestInfo, len(resp.Categories))
	c.JSON(http.StatusOK, resp.Categories)
}
