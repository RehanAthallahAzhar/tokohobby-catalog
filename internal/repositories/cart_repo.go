package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/models"
	customRedis "github.com/RehanAthallahAzhar/tokohobby-catalog/internal/pkg/redis"
)

type CartRepository interface {
	AddItem(ctx context.Context, userID, productID uuid.UUID, item models.RedisCartItem) error
	GetAllItems(ctx context.Context, userID uuid.UUID) (map[string]models.RedisCartItem, error)
	UpdateItem(ctx context.Context, userID, productID uuid.UUID, newQuantity int, newDescription string) error
	RemoveItem(ctx context.Context, userID, productID uuid.UUID) error
}

type cartRepositoryRedis struct {
	redisClient *customRedis.RedisClient
	log         *logrus.Logger
}

func NewCartRepository(redisClient *customRedis.RedisClient, log *logrus.Logger) CartRepository {
	return &cartRepositoryRedis{
		redisClient: redisClient,
		log:         log,
	}
}

func (r *cartRepositoryRedis) getCartKey(userID uuid.UUID) string {
	return fmt.Sprintf("cart:%s", userID.String())
}

func (r *cartRepositoryRedis) AddItem(ctx context.Context, userID, productID uuid.UUID, item models.RedisCartItem) error {
	cartKey := r.getCartKey(userID)

	itemJSON, err := json.Marshal(item)
	if err != nil {
		r.log.WithError(err).Error("Failed to marshal basket items")
		return fmt.Errorf("failed to process cart items: %w", err)
	}

	if err := r.redisClient.Client.HSet(ctx, cartKey, productID.String(), itemJSON).Err(); err != nil {
		r.log.WithError(err).Error("Failed to save item to Redis")
		return fmt.Errorf("failed to add item to cart: %w", err)
	}

	return nil
}

func (r *cartRepositoryRedis) GetAllItems(ctx context.Context, userID uuid.UUID) (map[string]models.RedisCartItem, error) {
	cartKey := r.getCartKey(userID)

	itemsMapStr, err := r.redisClient.Client.HGetAll(ctx, cartKey).Result()
	if err != nil {
		r.log.WithError(err).Error("Failed to retrieve basket from Redis")
		return nil, fmt.Errorf("failed to retrieve cart data: %w", err)
	}

	resultMap := make(map[string]models.RedisCartItem, len(itemsMapStr))
	for productID, itemJSON := range itemsMapStr {
		var item models.RedisCartItem
		if err := json.Unmarshal([]byte(itemJSON), &item); err != nil {
			r.log.WithField("product_id", productID).WithError(err).Warn("Failed to unmarshal basket item, item skipped")
			continue
		}
		resultMap[productID] = item
	}

	return resultMap, nil
}

func (r *cartRepositoryRedis) UpdateItem(ctx context.Context, userID, productID uuid.UUID, newQuantity int, newDescription string) error {
	cartKey := r.getCartKey(userID)
	productIDStr := productID.String()
	logger := r.log.WithFields(logrus.Fields{"cart_key": cartKey, "product_id": productIDStr})

	itemJSON, err := r.redisClient.Client.HGet(ctx, cartKey, productIDStr).Result()
	if err == redis.Nil {
		logger.Warn("Trying to update an item that is not in the cart")
		return fmt.Errorf("item not found in cart")
	}
	if err != nil {
		logger.WithError(err).Error("Failed to retrieve HGET item from Redis")
		return fmt.Errorf("failed to retrieve item from cart: %w", err)
	}

	var item models.RedisCartItem
	if err := json.Unmarshal([]byte(itemJSON), &item); err != nil {
		logger.WithError(err).Error("Failed to unmarshal basket items from Redis")
		return fmt.Errorf("corrupt basket data: %w", err)
	}

	item.Quantity = newQuantity
	item.Description = newDescription
	item.UpdatedAt = time.Now()

	updatedItemJSON, err := json.Marshal(item)
	if err != nil {
		logger.WithError(err).Error("Failed to marshal updated basket items")
		return fmt.Errorf("failed to process item update: %w", err)
	}

	if err := r.redisClient.Client.HSet(ctx, cartKey, productIDStr, updatedItemJSON).Err(); err != nil {
		logger.WithError(err).Error("Failed to update HSET item to Redis")
		return fmt.Errorf("failed to save updates to the cart: %w", err)
	}

	logger.Info("The quantity of items in Redis has been successfully updated.")
	return nil
}

func (r *cartRepositoryRedis) RemoveItem(ctx context.Context, userID, productID uuid.UUID) error {
	cartKey := r.getCartKey(userID)

	if err := r.redisClient.Client.HDel(ctx, cartKey, productID.String()).Err(); err != nil {
		r.log.WithError(err).Error("Failed to delete item from Redis")
		return fmt.Errorf("failed to remove item from cart: %w", err)
	}

	return nil
}
