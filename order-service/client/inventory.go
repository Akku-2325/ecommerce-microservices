package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"
)

var ErrProductNotFound = errors.New("product not found in inventory")
var ErrInventoryServiceUnavailable = errors.New("inventory service unavailable or returned an error")
var ErrInsufficientStock = errors.New("insufficient stock available")

type ProductInfo struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
	Stock int     `json:"stock"`
}

type InventoryClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewInventoryClient(baseURL string) *InventoryClient {
	return &InventoryClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second, // Set a reasonable timeout
		},
	}
}

func (c *InventoryClient) GetProductDetails(ctx context.Context, productID string) (*ProductInfo, error) {
	url := fmt.Sprintf("%s/products/%s", c.baseURL, productID)
	log.Printf("InventoryClient: Calling GET %s", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request to inventory service: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("InventoryClient: Error calling %s: %v", url, err)
		return nil, fmt.Errorf("%w: %v", ErrInventoryServiceUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		log.Printf("InventoryClient: Product %s not found (404)", productID)
		return nil, ErrProductNotFound
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("InventoryClient: Received non-OK status %d from %s", resp.StatusCode, url)
		return nil, fmt.Errorf("%w: status code %d", ErrInventoryServiceUnavailable, resp.StatusCode)
	}

	var productInfo ProductInfo
	if err := json.NewDecoder(resp.Body).Decode(&productInfo); err != nil {
		log.Printf("InventoryClient: Error decoding response from %s: %v", url, err)
		return nil, fmt.Errorf("failed to decode inventory service response: %w", err)
	}

	log.Printf("InventoryClient: Successfully fetched product %s: Price=%.2f, Stock=%d", productID, productInfo.Price, productInfo.Stock)
	return &productInfo, nil
}
