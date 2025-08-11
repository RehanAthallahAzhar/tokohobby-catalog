package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/RehanAthallahAzhar/shopeezy-inventory-cart/internal/db"
	"github.com/RehanAthallahAzhar/shopeezy-inventory-cart/internal/entities"
	"github.com/RehanAthallahAzhar/shopeezy-inventory-cart/internal/helpers"
	"github.com/RehanAthallahAzhar/shopeezy-inventory-cart/internal/models"
	"github.com/RehanAthallahAzhar/shopeezy-inventory-cart/internal/pkg/errors"
	"github.com/RehanAthallahAzhar/shopeezy-inventory-cart/internal/pkg/redisclient"
	"github.com/google/uuid"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

type contextKey string

const userIDKey contextKey = "userID"

type CartRepository interface {
	GetCartItemsByUserID(ctx context.Context, userID uuid.UUID) ([]models.CartItem, error)
	GetCartItemByProductID(ctx context.Context, userID, productID uuid.UUID) (*models.CartItem, error)
	// GetCartItemsByUsername(ctx context.Context, Username string) ([]models.CartItem, error) //coming soon
	AddItemToCart(ctx context.Context, userID, cartID uuid.UUID, cartData *entities.Cart) error
	UpdateItemQuantity(ctx context.Context, productID, userID uuid.UUID, req *entities.Cart) error
	RemoveItemFromCart(ctx context.Context, userID, productID uuid.UUID) error
	RestoreCartFromDB(ctx context.Context, userID uuid.UUID) error
	CheckoutCart(ctx context.Context, userID uuid.UUID) (*entities.Order, error)
}

type cartRepository struct {
	db          *db.Queries
	rawDB       *sql.DB
	productRepo ProductRepository
	redisClient *redisclient.RedisClient
}

func NewCartRepository(sqlcQueries *db.Queries, rawDB *sql.DB, productRepo ProductRepository, redisClient *redisclient.RedisClient) CartRepository {
	return &cartRepository{db: sqlcQueries, rawDB: rawDB, productRepo: productRepo, redisClient: redisClient}
}

func (r *cartRepository) getCartKey(userID string) string {
	return fmt.Sprintf("cart:%s", userID)
}

func (r *cartRepository) GetCartItemsByUserID(ctx context.Context, userID uuid.UUID) ([]models.CartItem, error) {

	cartKey := r.getCartKey(userID.String())

	uuidCartKey, err := helpers.StringToUUID(cartKey)
	if err != nil {
		return nil, fmt.Errorf("invalid cartKey: %w", err)
	}

	// Retrieve all fields and values from Redis Hash
	cartMap, err := r.redisClient.Client.HGetAll(ctx, cartKey).Result()
	if err != nil {
		return nil, errors.ErrCartRetrievalFail
	}

	var cartItems []models.CartItem
	for productID, jsonStr := range cartMap {
		var item models.RedisCartItem
		err := json.Unmarshal([]byte(jsonStr), &item)
		if err != nil {
			log.Printf("Warning: Failed to parse cart item %s: %v", productID, err)
			continue
		}

		uuidProductID, err := helpers.StringToUUID(productID)
		if err != nil {
			return nil, fmt.Errorf("invalid productID: %w", err)
		}

		// Retrieve product details from the (cached) ProductRepository
		product, err := r.productRepo.GetProductByID(ctx, uuidProductID)
		if err != nil {
			log.Printf("Warning: Failed to get product details %s for user's cart %s: %v", productID, userID, err)
			continue // Proceed to the next item if the product is not found
		}

		cartItems = append(cartItems, models.CartItem{
			ID:              uuidCartKey,
			SellerID:        product.SellerID,
			SellerName:      product.Name,
			Quantity:        item.Quantity,
			ProductID:       uuidProductID,
			ProductName:     product.Name,
			ProductPrice:    product.Price,
			CartDescription: item.CartDescription,
			CreatedAt:       item.CreatedAt,
			UpdatedAt:       item.UpdatedAt,
		})
	}
	return cartItems, nil
}

func (r *cartRepository) GetCartItemByProductID(ctx context.Context, userID, productID uuid.UUID) (*models.CartItem, error) {
	cartKey := r.getCartKey(userID.String())

	uuidCartKey, err := helpers.StringToUUID(cartKey)
	if err != nil {
		return nil, fmt.Errorf("invalid cartKey: %w", err)
	}

	jsonStr, err := r.redisClient.Client.HGet(ctx, cartKey, productID.String()).Result()
	if err == redis.Nil {
		return nil, errors.ErrCartItemNotFound
	} else if err != nil {
		return nil, fmt.Errorf("failed to get cart item from Redis: %w", err)
	}

	var redisItem models.RedisCartItem
	if err := json.Unmarshal([]byte(jsonStr), &redisItem); err != nil {
		return nil, fmt.Errorf("failed to parse cart item JSON: %w", err)
	}

	product, err := r.productRepo.GetProductByID(ctx, productID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrProductNotFound
		}
		return nil, fmt.Errorf("failed to retrieve product info: %w", err)
	}

	return &models.CartItem{
		ID:              uuidCartKey,
		SellerID:        product.SellerID,
		SellerName:      product.Name,
		Quantity:        redisItem.Quantity,
		ProductID:       productID,
		ProductName:     product.Name,
		ProductPrice:    product.Price,
		CartDescription: redisItem.CartDescription,
		CreatedAt:       redisItem.CreatedAt,
		UpdatedAt:       redisItem.UpdatedAt,
	}, nil
}

func (r *cartRepository) AddItemToCart(ctx context.Context, userID, cartID uuid.UUID, cartData *entities.Cart) error {
	cartKey := r.getCartKey(userID.String())

	currentStr, err := r.redisClient.Client.HGet(ctx, cartKey, cartData.ProductID.String()).Result()
	var currentQuantity int
	var existingItem *models.RedisCartItem

	if err == nil {
		var parsedItem models.RedisCartItem
		if err := json.Unmarshal([]byte(currentStr), &parsedItem); err == nil {
			currentQuantity = parsedItem.Quantity
			existingItem = &parsedItem
		}
	} else if err != redis.Nil {
		return fmt.Errorf("failed to retrieve an item from the Redis carts: %w", err)
	}

	newQuantity := currentQuantity + cartData.Quantity
	if newQuantity <= 0 {
		return r.RemoveItemFromCart(ctx, userID, cartData.ProductID)
	}

	cartRedis := NewRedisCartItem(newQuantity, cartData.Description, existingItem)

	cartRedisJSON, err := json.Marshal(cartRedis)
	if err != nil {
		return fmt.Errorf("failed to marshal cart item: %w", err)
	}

	// Store to Redis
	pipe := r.redisClient.Client.Pipeline()
	pipe.HSet(ctx, cartKey, cartData.ProductID, cartRedisJSON)
	pipe.Expire(ctx, cartKey, 24*time.Hour)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to add/update item to Redis cart: %w", err)
	}
	// Async backup ke database
	go func() {
		bgCtx := context.WithValue(context.Background(), userIDKey, userID)
		err := r.backupCartItemToDB(bgCtx, cartID, userID, cartData.ProductID, newQuantity, cartData.Description)
		if err != nil {
			log.Printf("failed to backup cart item to DB (user: %s, product: %s): %v", userID, cartData.ProductID, err)
		}
	}()

	log.Printf("User %s added product %s (qty: %d) to cart. Total: %d", userID, cartData.ProductID, cartData.Quantity, newQuantity)
	return nil
}

func (r *cartRepository) UpdateItemQuantity(ctx context.Context, productID, userID uuid.UUID, req *entities.Cart) error {
	if req.Quantity <= 0 {
		return errors.ErrInvalidRequestPayload
	}

	// Validate whether the product exists
	product, err := r.productRepo.GetProductByID(ctx, productID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.ErrCartNotFound
		}
		return fmt.Errorf("product not found: %w", err)
	}

	// Enough stock validation
	if product.Stock < req.Quantity {
		return errors.ErrInvalidRequestPayload
	}

	cartKey := r.getCartKey(userID.String())

	// Retrieve old items from Redis to maintain CreatedAt
	var existingItem *models.RedisCartItem
	currentStr, err := r.redisClient.Client.HGet(ctx, cartKey, productID.String()).Result()
	if err == nil {
		var parsedItem models.RedisCartItem
		if unmarshalErr := json.Unmarshal([]byte(currentStr), &parsedItem); unmarshalErr == nil {
			existingItem = &parsedItem
		}
	} else if err != redis.Nil {
		return fmt.Errorf("failed to retrieve existing cart item: %w", err)
	}

	// Create a new item with a new quantity and description
	newItem := NewRedisCartItem(req.Quantity, req.Description, existingItem)
	itemJSON, err := json.Marshal(newItem)
	if err != nil {
		return fmt.Errorf("failed to marshal updated cart item: %w", err)
	}

	// Save to Redis
	pipe := r.redisClient.Client.Pipeline()
	pipe.HSet(ctx, cartKey, productID, itemJSON)
	pipe.Expire(ctx, cartKey, 24*time.Hour)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to update Redis cart item: %w", err)
	}

	// Async backup to DB
	go func() {
		bgCtx := context.WithValue(context.Background(), userIDKey, userID)
		err := r.backupCartItemToDB(bgCtx, productID, userID, productID, req.Quantity, req.Description)
		if err != nil {
			log.Printf("failed to backup cart item to DB (user: %s, product: %s): %v", userID, productID, err)
		}
	}()

	log.Printf("User %s updated product %s to qty: %d", userID, productID, req.Quantity)
	return nil
}

func (r *cartRepository) RemoveItemFromCart(ctx context.Context, userID, productID uuid.UUID) error {
	cartKey := fmt.Sprintf("cart:%s", userID)

	if err := r.redisClient.Client.HDel(ctx, cartKey, productID.String()).Err(); err != nil {
		return fmt.Errorf("failed to remove item from Redis cart: %w", err)
	}

	err := r.db.DeleteCart(ctx, db.DeleteCartParams{
		ID:        userID,
		ProductID: productID,
	})

	if err != nil {
		log.Printf("warning: failed to delete cart item from DB for user %s and product %s: %v", userID, productID, err)
	}

	log.Printf("User %s removed product %s from cart", userID, productID)
	return nil
}

func (r *cartRepository) CheckoutCart(ctx context.Context, userID uuid.UUID) (*entities.Order, error) {
	// Get all items from the Redis cart
	cartItems, err := r.GetCartItemsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart items for checkout: %w", err)
	}

	if len(cartItems) == 0 {
		return nil, errors.ErrCartEmpty
	}

	// Start transaction
	tx, err := r.rawDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin tx: %w", err)
	}

	qtx := r.db.WithTx(tx)

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	// Buat order baru
	orderID := uuid.New()
	orderDate := time.Now()

	err = qtx.CreateOrder(ctx, db.CreateOrderParams{
		ID:        orderID,
		UserID:    userID,
		Status:    "Pending",
		OrderDate: orderDate,
		CreatedAt: orderDate,
		UpdatedAt: orderDate,
	})
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	var totalAmount int
	// Add OrderItems to the database and decrement stock
	for _, item := range cartItems {
		if err := qtx.CreateOrderItem(ctx, db.CreateOrderItemParams{
			ID:        uuid.New(),
			OrderID:   orderID,
			ProductID: item.ProductID,
			Quantity:  int32(item.Quantity),
			Price:     int32(item.ProductPrice),
		}); err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to create order item: %w", err)
		}

		// Decrement stock in the database
		if err := r.productRepo.DecrementProductStock(ctx, item.ProductID, item.Quantity); err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to decrement stock for product %s: %w", item.ProductName, err)
		}

		if err := r.RemoveItemFromCart(ctx, userID, item.ProductID); err != nil {
			log.Printf("Warning: Failed to clear user %s's cart from Redis after checkout: %v", userID, err)
		}

		totalAmount += item.ProductPrice * item.Quantity
	}

	err = qtx.UpdateOrderTotalAmount(ctx, db.UpdateOrderTotalAmountParams{
		ID:          orderID,
		TotalAmount: helpers.IntToNullInt32(totalAmount),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update order total: %w", err)
	}

	// 7. Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &entities.Order{
		ID:          orderID,
		UserID:      userID,
		Status:      "Pending",
		OrderDate:   time.Now(),
		TotalAmount: totalAmount,
	}, nil
}

func (r *cartRepository) backupCartItemToDB(ctx context.Context, cartID, userID, productID uuid.UUID, quantity int, desc string) error {
	_, err := r.db.GetCartItem(ctx, db.GetCartItemParams{
		UserID:    userID,
		ProductID: productID,
	})

	if err == gorm.ErrRecordNotFound {
		if err == sql.ErrNoRows {
			return r.db.InsertCartItem(ctx, db.InsertCartItemParams{
				ID:          cartID,
				UserID:      userID,
				ProductID:   productID,
				Quantity:    int32(quantity),
				Description: sql.NullString{String: desc, Valid: desc != ""},
			})
		}
		return err
	}

	return r.db.UpdateCartItem(ctx, db.UpdateCartItemParams{
		UserID:      userID,
		ProductID:   productID,
		Quantity:    int32(quantity),
		Description: sql.NullString{String: desc, Valid: desc != ""},
	})

}

func (r *cartRepository) RestoreCartFromDB(ctx context.Context, userID uuid.UUID) error {
	items, err := r.GetCartItemsByUserID(ctx, userID)
	if err != nil {
		return err
	}

	if len(items) == 0 {
		return nil
	}

	cartKey := r.getCartKey(userID.String())
	pipe := r.redisClient.Client.Pipeline()
	for _, item := range items {
		pipe.HSet(ctx, cartKey, item.ProductID, item.Quantity)
	}
	pipe.Expire(ctx, cartKey, 24*time.Hour)
	_, err = pipe.Exec(ctx)
	return err
}

func NewRedisCartItem(quantity int, description string, existing *models.RedisCartItem) *models.RedisCartItem {
	now := time.Now()

	item := &models.RedisCartItem{
		Quantity:        quantity,
		CartDescription: description,
		UpdatedAt:       now,
	}

	if existing != nil {
		item.CreatedAt = existing.CreatedAt
	} else {
		item.CreatedAt = now
	}

	return item
}
