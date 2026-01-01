package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/entities"
	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/helpers"
	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/models"
	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/pkg/grpc/account"
	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/pkg/redis"
	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/repositories"

	accountpb "github.com/RehanAthallahAzhar/tokohobby-protos/pb/account"
)

type CartSource interface {
	models.RedisCartItem
}

type CartService interface {
	AddItemToCart(ctx context.Context, userID, productID uuid.UUID, req *models.CartRequest) error
	GetCartItemsByUserID(ctx context.Context, userID uuid.UUID) (*entities.Cart, error)
	UpdateItem(ctx context.Context, userID, productID uuid.UUID, newQuantity int, newDescription string) error
	RemoveItemFromCart(ctx context.Context, userID, productID uuid.UUID) error
}

type cartServiceImpl struct {
	cartRepo      repositories.CartRepository
	productSvc    ProductService
	redisClient   *redis.RedisClient
	accountClient *account.AccountClient
	log           *logrus.Logger
}

func NewCartService(
	repo repositories.CartRepository,
	productSvc ProductService,
	redis *redis.RedisClient,
	accountClient *account.AccountClient,
	log *logrus.Logger,
) CartService {
	return &cartServiceImpl{
		cartRepo:      repo,
		productSvc:    productSvc,
		redisClient:   redis,
		accountClient: accountClient,
		log:           log,
	}
}

func (s *cartServiceImpl) AddItemToCart(ctx context.Context, userID, productID uuid.UUID, req *models.CartRequest) error {
	logger := s.log.WithFields(logrus.Fields{
		"user_id":    userID,
		"product_id": productID,
		"quantity":   req.Quantity,
	})
	logger.Info("Starting the process of adding items to the cart")

	if req.Quantity <= 0 {
		return fmt.Errorf("the quantity must be greater than 0")
	}

	item := models.RedisCartItem{
		Quantity:    req.Quantity,
		Description: req.Description,
		Checked:     true,
		AddedAt:     time.Now(),
	}

	log.Println(item)

	if err := s.cartRepo.AddItem(ctx, userID, productID, item); err != nil {
		logger.WithError(err).Error("Gagal saat memanggil repository untuk menambah item")
		return err
	}

	logger.Info("Item successfully added to cart")

	return nil
}

func (s *cartServiceImpl) GetCartItemsByUserID(ctx context.Context, userID uuid.UUID) (*entities.Cart, error) {
	logger := s.log.WithField("user_id", userID)
	logger.Info("Retrieving items from the user's cart")

	itemsMap, err := s.cartRepo.GetAllItems(ctx, userID)
	if err != nil {
		return nil, err
	}

	if len(itemsMap) == 0 {
		return &entities.Cart{
			UserID:     userID,
			Items:      []entities.CartItem{},
			TotalItems: 0,
		}, nil
	}

	var productIDs []uuid.UUID
	for idStr := range itemsMap {
		uuid, err := helpers.StringToUUID(idStr)
		if err != nil {
			return nil, fmt.Errorf("error converting string to UUID: %w", err)
		}
		productIDs = append(productIDs, uuid)
	}

	productsResponse, err := s.productSvc.GetProductByIDs(ctx, productIDs)
	if err != nil {
		logger.WithError(err).Error("Gagal mengambil detail produk dari Product Service")
		return nil, fmt.Errorf("gagal mengambil detail produk: %w", err)
	}
	productDetailsMap := make(map[string]*entities.Product)
	for _, p := range productsResponse {
		productDetailsMap[p.ID.String()] = &p
	}

	sellerIDMap := make(map[string]bool)
	for _, productID := range productIDs {
		productDetail, ok := productDetailsMap[productID.String()]
		if !ok {
			logger.WithField("product_id", productID.String()).Warn("Detail produk tidak ditemukan, item dilewati.")
			continue
		}

		sellerIDMap[productDetail.SellerID.String()] = true
	}

	var sellerIDs []string
	for sellerID := range sellerIDMap {
		sellerIDs = append(sellerIDs, sellerID)
	}

	// account
	accountDetailMap, err := s.fetchAccountDetails(ctx, sellerIDs)
	if err != nil {
		return nil, err
	}

	finalItems := make([]entities.CartItem, 0, len(itemsMap))
	for productIDStr, redisItem := range itemsMap {
		productDetail, ok := productDetailsMap[productIDStr]
		if !ok {
			logger.WithField("product_id", productIDStr).Warn("Detail produk tidak ditemukan, item dilewati.")
			continue
		}
		accountDetail, ok := accountDetailMap[productDetail.SellerID.String()]
		if !ok {
			logger.WithField("seller_id", productDetail.SellerID.String()).Warn("Detail penjual tidak ditemukan, item dilewati.")
			continue
		}

		productID, _ := uuid.Parse(productIDStr)
		sellerName := accountDetail.Name

		assembledItem := toDomainCartItem(productID, redisItem, productDetail, sellerName)
		finalItems = append(finalItems, *assembledItem)
	}

	finalCart := toDomainCart(userID, finalItems)

	logger.Info("Successfully retrieved and enriched the basket items")
	return finalCart, nil
}

func (s *cartServiceImpl) UpdateItem(ctx context.Context, userID, productID uuid.UUID, newQuantity int, newDescription string) error {
	logger := s.log.WithFields(logrus.Fields{"user_id": userID, "product_id": productID, "new_quantity": newQuantity})

	if userID == uuid.Nil || productID == uuid.Nil {
		return fmt.Errorf("invalid user ID or product ID")
	}

	if newQuantity == 0 {
		logger.Info("Quantity is 0, removing item from cart")
		return s.cartRepo.RemoveItem(ctx, userID, productID)
	}

	if newQuantity < 0 {
		return fmt.Errorf("kuantitas tidak boleh negatif")
	}

	logger.Info("Call Product Service for stock validation")

	productsSvc, err := s.productSvc.GetProductByID(ctx, productID)
	if err != nil {
		logger.WithError(err).Error("Failed to retrieve product details from Product Service")
		return fmt.Errorf("failed to retrieve product details: %w", err)
	}

	if int(productsSvc.Stock) < newQuantity {
		logger.Warnf("Stock is insufficient. Requested: %d, Available: %d", newQuantity, productsSvc.Stock)
		return fmt.Errorf("insufficient stock for product '%s'", productsSvc.Name)
	}

	return s.cartRepo.UpdateItem(ctx, userID, productID, newQuantity, newDescription)
}

func (s *cartServiceImpl) RemoveItemFromCart(ctx context.Context, userID, productID uuid.UUID) error {
	if userID == uuid.Nil || productID == uuid.Nil {
		return fmt.Errorf("invalid user ID or product ID")
	}

	logger := s.log.WithFields(logrus.Fields{
		"user_id":    userID,
		"product_id": productID,
	})
	logger.Info("Remove items from cart")

	return s.cartRepo.RemoveItem(ctx, userID, productID)
}

// ------- HELPERS -------

func (s *cartServiceImpl) fetchAccountDetails(ctx context.Context, sellerIDs []string) (map[string]*accountpb.User, error) {
	accountResponse, err := s.accountClient.GetUsers(ctx, sellerIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve seller details via gRPC: %w", err)
	}

	accountDetailsMap := make(map[string]*accountpb.User)
	for _, a := range accountResponse.Users {
		accountDetailsMap[a.Id] = a
	}

	return accountDetailsMap, nil
}

func toDomainCartItem(
	productID uuid.UUID,
	redisItem models.RedisCartItem,
	productDetail *entities.Product,
	sellerName string,
) *entities.CartItem {
	return &entities.CartItem{
		ProductID:       productID,
		ProductName:     productDetail.Name,
		ProductImageURL: "",
		Price:           float64(productDetail.Price),
		Stock:           int(productDetail.Stock),
		SellerID:        productDetail.SellerID,
		SellerName:      sellerName,
		Quantity:        redisItem.Quantity,
		Description:     redisItem.Description,
		Checked:         redisItem.Checked,
	}
}

func toDomainCart(userID uuid.UUID, items []entities.CartItem) *entities.Cart {
	var cartItems []entities.CartItem

	cartItems = append(cartItems, items...)

	return &entities.Cart{
		UserID:     userID,
		Items:      cartItems,
		TotalItems: len(cartItems),
	}
}
