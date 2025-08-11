package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/RehanAthallahAzhar/shopeezy-inventory-cart/internal/db"
	"github.com/RehanAthallahAzhar/shopeezy-inventory-cart/internal/entities"
	"github.com/RehanAthallahAzhar/shopeezy-inventory-cart/internal/helpers"
	"github.com/RehanAthallahAzhar/shopeezy-inventory-cart/internal/models"
	"github.com/RehanAthallahAzhar/shopeezy-inventory-cart/internal/pkg/redisclient"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProductRepository interface {
	GetAllProducts(ctx context.Context) ([]models.ProductWithSeller, error)
	GetProductByID(ctx context.Context, id uuid.UUID) (*models.ProductWithSeller, error)
	GetProductsBySellerID(ctx context.Context, id uuid.UUID) ([]models.ProductWithSeller, error)
	GetProductsByName(ctx context.Context, name string) ([]models.ProductWithSeller, error)
	CreateProduct(ctx context.Context, product *entities.Product) error
	UpdateProduct(ctx context.Context, product *entities.Product) error
	DeleteProduct(ctx context.Context, id uuid.UUID) error
	DecrementProductStock(ctx context.Context, productID uuid.UUID, quantity int) error
	IncrementProductStock(ctx context.Context, productID uuid.UUID, quantity int) error
	InvalidateProductCache(ctx context.Context, productID uuid.UUID)
}

type productRepository struct {
	db          *db.Queries
	redisClient *redisclient.RedisClient
}

func NewProductRepository(sqlcQueries *db.Queries, redisClient *redisclient.RedisClient) ProductRepository {
	return &productRepository{
		db:          sqlcQueries,
		redisClient: redisClient,
	}
}

func (r *productRepository) GetAllProducts(ctx context.Context) ([]models.ProductWithSeller, error) {
	cacheKey := "all_products"
	var products []models.ProductWithSeller

	if val, err := r.redisClient.Client.Get(ctx, cacheKey).Result(); err == nil {
		if err := json.Unmarshal([]byte(val), &products); err == nil {
			log.Println("GetAllProducts from Redis")
			return products, nil
		}
	}

	rows, err := r.db.GetAllProducts(ctx)
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		products = append(products, models.ProductWithSeller{
			ID:          row.ID,
			SellerID:    row.SellerID,
			SellerName:  row.SellerName.String,
			Name:        row.Name,
			Price:       int(row.Price),
			Stock:       int(row.Stock),
			Discount:    int(row.Discount.Int32),
			Type:        row.Type.String,
			Description: row.Description.String,
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		})
	}

	if jsonBytes, err := json.Marshal(products); err == nil {
		_ = r.redisClient.Client.Set(ctx, cacheKey, jsonBytes, 2*time.Minute).Err()
	}

	return products, nil
}

func (r *productRepository) GetProductByID(ctx context.Context, id uuid.UUID) (*models.ProductWithSeller, error) {
	cacheKey := fmt.Sprintf("product:%s", id)
	var product models.ProductWithSeller

	if val, err := r.redisClient.Client.Get(ctx, cacheKey).Result(); err == nil {
		if err := json.Unmarshal([]byte(val), &product); err == nil {
			log.Println("Product retrieved from Redis")
			return &product, nil
		}
	}

	row, err := r.db.GetProductByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("product not found: %w", err)
	}

	product = models.ProductWithSeller{
		ID:          row.ID,
		SellerID:    row.SellerID,
		SellerName:  row.SellerName.String,
		Name:        row.Name,
		Price:       int(row.Price),
		Stock:       int(row.Stock),
		Discount:    int(row.Discount.Int32),
		Type:        row.Type.String,
		Description: row.Description.String,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}

	if jsonBytes, err := json.Marshal(product); err == nil {
		_ = r.redisClient.Client.Set(ctx, cacheKey, jsonBytes, 5*time.Minute).Err()
	}

	return &product, nil
}

func (r *productRepository) GetProductsBySellerID(ctx context.Context, sellerID uuid.UUID) ([]models.ProductWithSeller, error) {
	sellerProductKey := fmt.Sprintf("products_by_seller:%s", sellerID)
	var products []models.ProductWithSeller

	val, err := r.redisClient.Client.Get(ctx, sellerProductKey).Result()
	if err == nil {
		if err := json.Unmarshal([]byte(val), &products); err == nil {
			log.Printf("Products by seller %s retrieved from Redis cache", sellerID)
			return products, nil
		}
		log.Printf("Error unmarshalling seller products from Redis: %v", err)
	} else if err != redis.Nil {
		log.Printf("Error accessing Redis for seller products: %v", err)
	}

	rows, err := r.db.GetProductsBySellerID(ctx, sellerID)
	if err != nil {
		return nil, fmt.Errorf("seller not found: %w", err)
	}

	for _, row := range rows {
		products = append(products, models.ProductWithSeller{
			ID:          row.ID,
			SellerID:    row.SellerID,
			SellerName:  row.SellerName.String,
			Name:        row.Name,
			Price:       int(row.Price),
			Stock:       int(row.Stock),
			Discount:    int(row.Discount.Int32),
			Type:        row.Type.String,
			Description: row.Description.String,
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		})
	}

	productsJSON, err := json.Marshal(products)
	if err != nil {
		log.Printf("Failed to marshal products for seller %s: %v", sellerID, err)
	} else {
		err = r.redisClient.Client.Set(ctx, sellerProductKey, productsJSON, 5*time.Minute).Err()
		if err != nil {
			log.Printf("Failed to cache products for seller %s: %v", sellerID, err)
		}
	}

	return products, nil
}

func (r *productRepository) GetProductsByName(ctx context.Context, name string) ([]models.ProductWithSeller, error) {
	sellerProductKey := fmt.Sprintf("products_by_name:%s", name)
	var products []models.ProductWithSeller

	val, err := r.redisClient.Client.Get(ctx, sellerProductKey).Result()
	if err == nil {
		if err := json.Unmarshal([]byte(val), &products); err == nil {
			log.Printf("Products by seller %s retrieved from Redis cache", name)
			return products, nil
		}
		log.Printf("Error unmarshalling seller products from Redis: %v", err)
	} else if err != redis.Nil {
		log.Printf("Error accessing Redis for seller products: %v", err)
	}

	rows, err := r.db.GetProductsByName(ctx, name)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("seller not found")
		}
		return nil, err
	}

	for _, row := range rows {
		products = append(products, models.ProductWithSeller{
			ID:          row.ID,
			SellerID:    row.SellerID,
			SellerName:  row.SellerName.String,
			Name:        row.Name,
			Price:       int(row.Price),
			Stock:       int(row.Stock),
			Discount:    int(row.Discount.Int32),
			Type:        row.Type.String,
			Description: row.Description.String,
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		})
	}

	productsJSON, err := json.Marshal(products)
	if err != nil {
		log.Printf("Failed to marshal products for seller %s: %v", name, err)
	} else {
		err = r.redisClient.Client.Set(ctx, sellerProductKey, productsJSON, 5*time.Minute).Err()
		if err != nil {
			log.Printf("Failed to cache products for seller %s: %v", name, err)
		}
	}

	return products, nil
}

func (r *productRepository) InvalidateProductCache(ctx context.Context, productID uuid.UUID) {
	productKey := fmt.Sprintf("product:%s", productID)
	allProductsKey := "all_products"

	// Clear specific product cache
	err := r.redisClient.Client.Del(ctx, productKey).Err()
	if err != nil {
		log.Printf("Failed to clear the product cache %s: %v", productID, err)
	} else {
		log.Printf("Product cache %s cleared.", productID)
	}

	// Clear cache of all products (cause the list has changed)
	err = r.redisClient.Client.Del(ctx, allProductsKey).Err()
	if err != nil {
		log.Printf("Failed to clear the all products cache: %v", err)
	} else {
		log.Println("Cache of all products cleared.")
	}
}

func (r *productRepository) CreateProduct(ctx context.Context, product *entities.Product) error {

	_, err := r.db.InsertProduct(ctx, db.InsertProductParams{
		ID:          product.ID,
		SellerID:    product.SellerID,
		Name:        product.Name,
		Price:       int32(product.Price),
		Stock:       int32(product.Stock),
		Discount:    helpers.IntToNullInt32(product.Discount),
		Type:        helpers.StringToNullString(product.Type),
		Description: helpers.StringToNullString(product.Description),
	})

	if err == nil {
		r.InvalidateProductCache(ctx, product.ID)
	}

	return err
}

func (r *productRepository) UpdateProduct(ctx context.Context, product *entities.Product) error {

	err := r.db.UpdateProduct(ctx, db.UpdateProductParams{
		ID:          product.ID,
		SellerID:    product.SellerID,
		Name:        product.Name,
		Price:       int32(product.Price),
		Stock:       int32(product.Stock),
		Discount:    helpers.IntToNullInt32(product.Discount),
		Type:        helpers.StringToNullString(product.Type),
		Description: helpers.StringToNullString(product.Description),
	})

	if err != nil {
		return err
	}

	r.InvalidateProductCache(ctx, product.ID)

	return nil
}

func (r *productRepository) DeleteProduct(ctx context.Context, id uuid.UUID) error {

	err := r.db.DeleteProduct(ctx, id)
	if err != nil {
		return err
	}

	r.InvalidateProductCache(ctx, id)

	return nil
}

func (r *productRepository) DecrementProductStock(ctx context.Context, productID uuid.UUID, quantity int) error {

	// Ambil produk dari database
	row, err := r.db.GetProductByID(ctx, productID)
	if err != nil {
		return fmt.Errorf("failed to fetch product: %w", err)
	}

	if int(row.Stock) < quantity {
		return fmt.Errorf("stok tidak mencukupi")
	}

	newStock := row.Stock - int32(quantity)

	// Update stok
	err = r.db.UpdateProductStock(ctx, db.UpdateProductStockParams{
		ID:    productID,
		Stock: newStock,
	})
	if err != nil {
		return fmt.Errorf("failed to update stock: %w", err)
	}

	r.InvalidateProductCache(ctx, productID)
	return nil
}

func (r *productRepository) IncrementProductStock(ctx context.Context, productID uuid.UUID, quantity int) error {
	row, err := r.db.GetProductByID(ctx, productID)
	if err != nil {
		return fmt.Errorf("failed to fetch product: %w", err)
	}

	newStock := row.Stock + int32(quantity)

	err = r.db.UpdateProductStock(ctx, db.UpdateProductStockParams{
		ID:    productID,
		Stock: newStock,
	})
	if err != nil {
		return fmt.Errorf("failed to update stock: %w", err)
	}

	r.InvalidateProductCache(ctx, productID)
	return nil
}
