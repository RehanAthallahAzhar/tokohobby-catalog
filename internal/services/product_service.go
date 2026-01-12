package services

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/entities"
	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/helpers"
	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/models"
	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/pkg/db"
	apperrors "github.com/RehanAthallahAzhar/tokohobby-catalog/internal/pkg/errors"
	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/pkg/redis"
	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/repositories"

	productpb "github.com/RehanAthallahAzhar/tokohobby-protos/pb/product"
)

type ProductSource interface {
	db.Product |
		db.GetAllProductsRow |
		db.GetProductsBySellerIDRow |
		db.GetProductsByNameRow |
		db.GetProductByIDRow |
		db.GetProductByIDsRow |
		db.GetProductsByTypeRow
}

type ProductService interface {
	CreateProduct(ctx context.Context, userID uuid.UUID, req *models.ProductRequest) (*entities.Product, error)
	GetAllProducts(ctx context.Context) ([]entities.Product, error)
	GetProductsBySellerID(ctx context.Context, sellerID uuid.UUID) ([]entities.Product, error)
	GetProductsByName(ctx context.Context, name string) ([]entities.Product, error)
	GetProductsByType(ctx context.Context, productType string) ([]entities.Product, error)
	GetProductByID(ctx context.Context, id uuid.UUID) (*entities.Product, error)
	GetProductByIDs(ctx context.Context, ids []uuid.UUID) ([]entities.Product, error)
	UpdateProduct(ctx context.Context, req *models.ProductRequest, productID, sellerID uuid.UUID, role string) (*entities.Product, error)
	DeleteProduct(ctx context.Context, productID, sellerID uuid.UUID, role string) (*entities.Product, error)
	ResetAllProductCaches(ctx context.Context) error
	DecreaseStock(ctx context.Context, items []*productpb.StockItem) ([]*entities.Product, error)
	IncreaseStock(ctx context.Context, items []*productpb.StockItem) ([]*entities.Product, error)
}

type productServiceImpl struct {
	productRepo    repositories.ProductRepository
	redisClient    *redis.RedisClient
	eventPublisher *validator.Validate
	validator      *validator.Validate
	log            *logrus.Logger
}

func NewProductService(
	productRepo repositories.ProductRepository,
	redisClient *redis.RedisClient,
	validator *validator.Validate,
	log *logrus.Logger,
) ProductService {
	return &productServiceImpl{
		productRepo: productRepo,
		redisClient: redisClient,
		validator:   validator,
		log:         log,
	}
}

func (s *productServiceImpl) CreateProduct(ctx context.Context, userID uuid.UUID, req *models.ProductRequest) (*entities.Product, error) {
	if err := s.validator.Struct(req); err != nil {
		validationErrors := err.(validator.ValidationErrors)

		var errorMessages []string
		for _, fieldErr := range validationErrors {
			errorMessages = append(errorMessages, fmt.Sprintf("Field '%s' failed on the '%s' tag", fieldErr.Field(), fieldErr.Tag()))
		}

		return nil, fmt.Errorf("%w: %s", apperrors.ErrInvalidRequestPayload, strings.Join(errorMessages, ", "))
	}

	product := &db.InsertProductParams{
		ID:          helpers.GenerateNewID(),
		SellerID:    userID,
		Name:        req.Name,
		Price:       int32(req.Price),
		Stock:       int32(req.Stock),
		Discount:    helpers.IntToNullInt32(req.Discount),
		Type:        helpers.StringToNullString(req.Type),
		Description: helpers.StringToNullString(req.Description),
	}

	dbProduct, err := s.productRepo.CreateProduct(ctx, product)
	if err != nil {
		return nil, fmt.Errorf("service: failed to add product: %w", err)
	}

	if err := s.InvalidateProductCache(ctx, dbProduct.ID); err != nil {
		s.log.Errorf("Failed to clear product cache: %v", err)
	}

	return toDomainProduct(dbProduct), nil
}

func (s *productServiceImpl) GetAllProducts(ctx context.Context) ([]entities.Product, error) {
	var products []entities.Product
	cacheKey := "all_products"

	if val, err := s.redisClient.Client.Get(ctx, cacheKey).Result(); err == nil {
		if err := json.Unmarshal([]byte(val), &products); err == nil {
			s.log.Info("Hit Cache untuk GetAllProducts")
			return products, nil
		}
	}

	dbProducts, err := s.productRepo.GetAllProducts(ctx)
	if err != nil {
		return nil, fmt.Errorf("service: failed to retrieve all products: %w", err)
	}

	domainProduct := toDomainProducts(dbProducts)

	jsonBytes, err := json.Marshal(domainProduct)
	if err == nil {
		if err := s.redisClient.Client.Set(ctx, cacheKey, jsonBytes, 5*time.Minute).Err(); err != nil {
			s.log.WithField("key", cacheKey).Warn("Failed to set cache")
		}
	}

	return domainProduct, nil
}

func (s *productServiceImpl) GetProductsBySellerID(ctx context.Context, sellerID uuid.UUID) ([]entities.Product, error) {
	var products []entities.Product
	cacheKey := fmt.Sprintf("products_by_seller:%s", sellerID)

	if val, err := s.redisClient.Client.Get(ctx, cacheKey).Result(); err == nil {
		if err := json.Unmarshal([]byte(val), &products); err == nil {
			s.log.WithField("seller_id", sellerID).Info("Hit Cache untuk GetProductsBySellerID")
			return products, nil
		}
	}

	dbProducts, err := s.productRepo.GetProductsBySellerID(ctx, sellerID)
	if err != nil {
		return nil, fmt.Errorf("service: failed to retrieve products by seller ID %s: %w", sellerID, err)
	}

	domainProduct := toDomainProducts(dbProducts)

	jsonBytes, err := json.Marshal(domainProduct)
	if err == nil {
		if err := s.redisClient.Client.Set(ctx, cacheKey, jsonBytes, 5*time.Minute).Err(); err != nil {
			s.log.WithField("key", cacheKey).Warn("Failed to set cache")
		}
	}

	return domainProduct, nil
}

func (s *productServiceImpl) GetProductsByName(ctx context.Context, name string) ([]entities.Product, error) {
	var products []entities.Product
	cacheKey := fmt.Sprintf("products_by_name:%s", name)

	if val, err := s.redisClient.Client.Get(ctx, cacheKey).Result(); err == nil {
		if err := json.Unmarshal([]byte(val), &products); err == nil {
			s.log.WithField("name", name).Info("Hit Cache untuk GetProductsByName")
			return products, nil
		}
	}

	dbProducts, err := s.productRepo.GetProductsByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("service: failed to retrieve products by name %s: %w", name, err)
	}

	if len(dbProducts) == 0 {
		return []entities.Product{}, nil
	}

	domainProducts := toDomainProducts(dbProducts)

	go func() {
		jsonBytes, err := json.Marshal(domainProducts)
		if err != nil {
			s.log.Errorf("Failed to marshal products for caching: %v", err)
			return
		}

		cacheCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if err := s.redisClient.Client.Set(cacheCtx, cacheKey, jsonBytes, 5*time.Minute).Err(); err != nil {
			s.log.Warnf("Failed to set cache in background for key '%s': %v", cacheKey, err)
		}
	}()

	return domainProducts, nil
}

func (s *productServiceImpl) GetProductsByType(ctx context.Context, productType string) ([]entities.Product, error) {
	var products []entities.Product
	cacheKey := fmt.Sprintf("products_by_type:%s", productType)

	if val, err := s.redisClient.Client.Get(ctx, cacheKey).Result(); err == nil {
		if err := json.Unmarshal([]byte(val), &products); err == nil {
			s.log.WithField("name", productType).Info("Hit Cache untuk GetProductsByName")
			return products, nil
		}
	}

	dbProducts, err := s.productRepo.GetProductsByType(ctx, productType)
	if err != nil {
		return nil, fmt.Errorf("service: failed to retrieve products by name %s: %w", productType, err)
	}

	if len(dbProducts) == 0 {
		return []entities.Product{}, nil
	}

	domainProducts := toDomainProducts(dbProducts)

	go func() {
		jsonBytes, err := json.Marshal(domainProducts)
		if err != nil {
			s.log.Errorf("Failed to marshal products for caching: %v", err)
			return
		}

		cacheCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if err := s.redisClient.Client.Set(cacheCtx, cacheKey, jsonBytes, 5*time.Minute).Err(); err != nil {
			s.log.Warnf("Failed to set cache in background for key '%s': %v", cacheKey, err)
		}
	}()

	return domainProducts, nil
}

func (s *productServiceImpl) GetProductByID(ctx context.Context, id uuid.UUID) (*entities.Product, error) {
	var products *entities.Product
	cacheKey := fmt.Sprintf("product:%s", id)

	if val, err := s.redisClient.Client.Get(ctx, cacheKey).Result(); err == nil {
		if err = json.Unmarshal([]byte(val), &products); err == nil {
			s.log.WithField("product_id", id).Info("Hit Cache untuk GetProductByID")
			return products, nil
		}
	}

	dbProduct, err := s.productRepo.GetProductByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("service: failed to retrieve product by ID: %w", err)
	}

	if dbProduct == nil {
		return &entities.Product{}, nil
	}

	domainProduct := toDomainProduct(dbProduct)

	jsonBytes, err := json.Marshal(domainProduct)
	if err == nil {
		if err = s.redisClient.Client.Set(ctx, cacheKey, jsonBytes, 5*time.Minute).Err(); err != nil {
			s.log.WithField("key", cacheKey).Warn("Failed to set cache")
		}

	}

	return domainProduct, nil
}

func (s *productServiceImpl) GetProductByIDs(ctx context.Context, ids []uuid.UUID) ([]entities.Product, error) {
	if len(ids) == 0 {
		return []entities.Product{}, nil
	}

	finalProducts := make([]entities.Product, 0, len(ids))
	cacheKeys := make([]string, len(ids))
	missedIDs := make(map[uuid.UUID]bool)

	for i, id := range ids {
		cacheKeys[i] = fmt.Sprintf("product:%s", id.String())
		missedIDs[id] = true
	}

	s.log.WithField("keys", cacheKeys).Info("Attempting to retrieve a product from the Redis cache with MGET")

	cachedResults, err := s.redisClient.Client.MGet(ctx, cacheKeys...).Result()
	if err != nil {
		s.log.WithError(err).Warn("Failed to run MGET Redis. Will retrieve everything from the database..")
	} else {
		for i, result := range cachedResults {
			if result != nil {
				if val, ok := result.(string); ok {
					var product entities.Product
					if json.Unmarshal([]byte(val), &product) == nil {
						finalProducts = append(finalProducts, product)
						delete(missedIDs, ids[i])
						s.log.WithField("product_id", ids[i]).Debug("Cache HIT untuk produk")
					}
				}
			}
		}
	}

	if len(missedIDs) > 0 {
		missedIDSlice := make([]uuid.UUID, 0, len(missedIDs))
		for id := range missedIDs {
			missedIDSlice = append(missedIDSlice, id)
		}

		s.log.WithField("missed_ids", missedIDSlice).Info("Cache MISS. Mengambil produk yang hilang dari database.")

		dbProducts, err := s.productRepo.GetProductByIDs(ctx, missedIDSlice)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve product from database: %w", err)
		}

		domainProducts := toDomainProducts(dbProducts)

		finalProducts = append(finalProducts, domainProducts...)

		if len(domainProducts) > 0 {
			pairs := make([]interface{}, 0, len(domainProducts)*2)
			for _, product := range domainProducts {
				cacheKey := fmt.Sprintf("product:%s", product.ID.String())
				jsonBytes, err := json.Marshal(product)
				if err == nil {
					pairs = append(pairs, cacheKey, string(jsonBytes))
				}
			}

			if len(pairs) > 0 {
				if err := s.redisClient.Client.MSet(ctx, pairs...).Err(); err != nil {
					s.log.WithError(err).Error("Failed to run MSET to save new products to the Redis cache")
				} else {
					pipe := s.redisClient.Client.Pipeline()
					ttl := 10 * time.Minute
					for i := 0; i < len(pairs); i += 2 {
						key := pairs[i].(string)
						pipe.Expire(ctx, key, ttl)
					}
					if _, err := pipe.Exec(ctx); err != nil {
						s.log.WithError(err).Error("Failed to set TTL for new product keys in the Redis cache")
					}
				}
			}
		}
	}

	return finalProducts, nil
}
func (s *productServiceImpl) UpdateProduct(ctx context.Context, req *models.ProductRequest, productID, sellerID uuid.UUID, role string) (*entities.Product, error) {
	if err := s.validator.Struct(req); err != nil {
		validationErrors := err.(validator.ValidationErrors)
		var errorMessages []string
		for _, fieldErr := range validationErrors {
			errorMessages = append(errorMessages, fmt.Sprintf("Field '%s' failed on the '%s' tag", fieldErr.Field(), fieldErr.Tag()))
		}
		return nil, fmt.Errorf("%w: %s", apperrors.ErrInvalidRequestPayload, strings.Join(errorMessages, ", "))
	}

	existingProduct, err := s.productRepo.GetProductByID(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("service: failed to find product for update: %w", err)
	}

	if role != "admin" && existingProduct.SellerID != sellerID {
		return nil, fmt.Errorf("service: product does not belong to this seller")
	}

	productParam := &db.UpdateProductParams{
		ID:          productID,
		SellerID:    existingProduct.SellerID,
		Name:        req.Name,
		Price:       int32(req.Price),
		Stock:       int32(req.Stock),
		Discount:    helpers.IntToNullInt32(req.Discount),
		Type:        helpers.StringToNullString(req.Type),
		Description: helpers.StringToNullString(req.Description),
	}

	dbProduct, err := s.productRepo.UpdateProduct(ctx, productParam)
	if err != nil {
		return nil, fmt.Errorf("service: failed to update product: %w", err)
	}

	if err := s.InvalidateProductCache(ctx, productID); err != nil {
		s.log.Errorf("Failed to clear product cache: %v", err)
	}

	return toDomainProduct(dbProduct), nil
}

func (s *productServiceImpl) DeleteProduct(ctx context.Context, productID, sellerID uuid.UUID, role string) (*entities.Product, error) {
	existingProduct, err := s.productRepo.GetProductByID(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("service: failed to find product for deletion: %w", err)
	}

	if role != "admin" && existingProduct.SellerID != sellerID {
		return nil, fmt.Errorf("service: product does not belong to this seller")
	}

	dbPproduct, err := s.productRepo.DeleteProduct(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("service: failed to delete product: %w", err)
	}

	if err := s.InvalidateProductCache(ctx, productID); err != nil {
		s.log.Errorf("Failed to clear product cache: %v", err)
	}

	return toDomainProduct(dbPproduct), nil
}

func (s *productServiceImpl) DecreaseStock(ctx context.Context, items []*productpb.StockItem) ([]*entities.Product, error) {
	tx, err := s.productRepo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Rollback

	updatedProducts := make([]*entities.Product, 0, len(items))

	for _, item := range items {
		productID, _ := uuid.Parse(item.ProductId)

		dbProduct, err := s.productRepo.DecreaseProductStock(ctx, tx, productID, item.QuantityToDecrease)
		if err != nil {
			return nil, fmt.Errorf("failed to process stock for product %s: %w", item.ProductId, err) // Rollback
		}

		updatedProducts = append(updatedProducts, toDomainProduct(dbProduct))
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit stock update transaction: %w", err)
	}

	go s.InvalidateCachesAfterUpdate(ctx, updatedProducts)

	return updatedProducts, nil
}

func (s *productServiceImpl) IncreaseStock(ctx context.Context, items []*productpb.StockItem) ([]*entities.Product, error) {
	tx, err := s.productRepo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	updatedProducts := make([]*entities.Product, 0, len(items))
	for _, item := range items {
		productID, _ := uuid.Parse(item.ProductId)
		params := db.IncreaseProductStockParams{
			ProductID:          productID,
			QuantityToIncrease: item.QuantityToDecrease,
		}

		dbProduct, err := s.productRepo.IncreaseProductStock(ctx, tx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to process stock increase for %s: %w", item.ProductId, err)
		}
		updatedProducts = append(updatedProducts, toDomainProduct(&dbProduct))
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	go s.InvalidateCachesAfterUpdate(ctx, updatedProducts)
	return updatedProducts, nil
}

// ------- HELPERS -------
func toDomainProduct[T ProductSource](dbProduct *T) *entities.Product {
	v := reflect.ValueOf(dbProduct)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	id := v.FieldByName("ID").Interface().(uuid.UUID)

	return &entities.Product{
		ID:          id,
		SellerID:    v.FieldByName("SellerID").Interface().(uuid.UUID),
		Name:        v.FieldByName("Name").Interface().(string),
		Price:       helpers.ConvertNullInt32(v.FieldByName("Price")),
		Stock:       helpers.ConvertNullInt32(v.FieldByName("Stock")),
		Discount:    helpers.ConvertNullInt32(v.FieldByName("Discount")),
		Type:        helpers.ConvertNullString(v.FieldByName("Type")),
		Description: helpers.ConvertNullString(v.FieldByName("Description")),
		CreatedAt:   v.FieldByName("CreatedAt").Interface().(time.Time),
		UpdatedAt:   v.FieldByName("UpdatedAt").Interface().(time.Time),
	}
}

func toDomainProducts[T ProductSource](dbProducts []T) []entities.Product {
	products := make([]entities.Product, 0, len(dbProducts))

	for _, dbProduct := range dbProducts {
		products = append(products, *toDomainProduct(&dbProduct))
	}

	return products
}

func (s *productServiceImpl) InvalidateProductCache(ctx context.Context, productID uuid.UUID) error {
	s.log.Infof("Invalidating caches for product %s and product list...", productID)

	keysToDelete := []string{
		fmt.Sprintf("product:%s", productID),
		"all_products",
	}

	err := s.redisClient.Client.Del(ctx, keysToDelete...).Err()
	if err != nil {
		s.log.Errorf("Failed to invalidate product cache keys (%v): %v", keysToDelete, err)
		return err
	}

	s.log.Infof("Cache keys %v successfully invalidated.", keysToDelete)
	return nil
}

func (s *productServiceImpl) ResetAllProductCaches(ctx context.Context) error {
	s.log.Info("Starting to reset ALL product caches...")

	if err := s.redisClient.Client.Del(ctx, "all_products").Err(); err != nil {
		s.log.Errorf("Failed to delete main product list cache: %v", err)
	}

	var cursor uint64
	pattern := "product:*"

	pipe := s.redisClient.Client.Pipeline()
	keysFound := 0

	for {
		keys, nextCursor, err := s.redisClient.Client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			s.log.Errorf("Error during Redis SCAN with pattern '%s': %v", pattern, err)
			return err
		}

		if len(keys) > 0 {
			pipe.Del(ctx, keys...)
			keysFound += len(keys)
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	if _, err := pipe.Exec(ctx); err != nil {
		s.log.Errorf("Failed to execute pipeline for deleting %d keys: %v", keysFound, err)
		return err
	}

	s.log.Infof("Successfully reset %d individual product caches and the main list cache.", keysFound)
	return nil
}

func (s *productServiceImpl) InvalidateCachesAfterUpdate(ctx context.Context, updatedProducts []*entities.Product) {
	s.log.Info("Invalidating product caches after stock update...")

	keysToDel := []string{"all_products"}

	for _, p := range updatedProducts {
		keysToDel = append(keysToDel, fmt.Sprintf("product:%s", p.ID.String()))
	}

	cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.redisClient.Client.Del(cacheCtx, keysToDel...).Err(); err != nil {
		s.log.Warnf("Failed to invalidate product caches: %v", err)
	} else {
		s.log.Infof("Successfully invalidated %d cache keys.", len(keysToDel))
	}
}
