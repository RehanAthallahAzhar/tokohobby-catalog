package services

import (
	"context"
	"encoding/json"
	stdErrors "errors"
	"fmt"
	"log"

	"github.com/RehanAthallahAzhar/shopeezy-inventory-cart/internal/entities"
	"github.com/RehanAthallahAzhar/shopeezy-inventory-cart/internal/helpers"
	"github.com/RehanAthallahAzhar/shopeezy-inventory-cart/internal/models"
	"github.com/RehanAthallahAzhar/shopeezy-inventory-cart/internal/models/converters"
	"github.com/RehanAthallahAzhar/shopeezy-inventory-cart/internal/pkg/errors"
	"github.com/RehanAthallahAzhar/shopeezy-inventory-cart/internal/pkg/rabbitmq"
	"github.com/RehanAthallahAzhar/shopeezy-inventory-cart/internal/repositories"
	"github.com/google/uuid"
)

type CartService interface {
	GetCartItemsByUserID(ctx context.Context, userID uuid.UUID) ([]models.CartItemResponse, error)
	GetCartItemByProductID(ctx context.Context, userID, productID uuid.UUID) (*models.CartItemResponse, error)
	AddItemToCart(ctx context.Context, userID uuid.UUID, cartData *models.CartRequest) ([]models.CartItemResponse, error)
	UpdateItemQuantity(ctx context.Context, productID, userID uuid.UUID, req *models.UpdateCartRequest) ([]models.CartItemResponse, error)
	RemoveItemFromCart(ctx context.Context, userID, productID uuid.UUID) error
	RestoreCartFromDB(ctx context.Context, userID uuid.UUID) ([]models.CartItemResponse, error)
	CheckoutCart(ctx context.Context, userID uuid.UUID) error
}

type cartServiceImpl struct {
	cartRepo       repositories.CartRepository
	productRepo    repositories.ProductRepository
	rabbitmqClient *rabbitmq.RabbitMQClient // Tambahkan RabbitMQClient
}

func NewCartService(cartRepo repositories.CartRepository, productRepo repositories.ProductRepository, rabbitmqClient *rabbitmq.RabbitMQClient) CartService {
	return &cartServiceImpl{
		cartRepo:       cartRepo,
		productRepo:    productRepo,
		rabbitmqClient: rabbitmqClient,
	}
}

func (s *cartServiceImpl) GetCartItemsByUserID(ctx context.Context, userID uuid.UUID) ([]models.CartItemResponse, error) {
	cartItems, err := s.cartRepo.GetCartItemsByUserID(ctx, userID)
	if err != nil {
		if stdErrors.Is(err, errors.ErrCartNotFound) {
			return nil, errors.ErrCartNotFound
		}
		log.Printf("Error in service getting cart for user %s: %v", userID, err)
		return nil, err
	}

	if len(cartItems) == 0 {
		return nil, errors.ErrProductNotFound
	}

	var res []models.CartItemResponse
	for _, cartItem := range cartItems {
		res = append(res, *converters.MapToCartResponse(userID, &cartItem))
	}

	return res, nil
}

func (s *cartServiceImpl) GetCartItemByProductID(ctx context.Context, userID, productID uuid.UUID) (*models.CartItemResponse, error) {
	cartItems, err := s.cartRepo.GetCartItemByProductID(ctx, userID, productID)
	if err != nil {
		if stdErrors.Is(err, errors.ErrCartNotFound) {
			return nil, errors.ErrCartNotFound
		}
		log.Printf("Error in service getting cart for user %s: %v", productID, err)
		return nil, fmt.Errorf("failed to get cart items: %w", err)
	}

	return converters.MapToCartResponse(userID, cartItems), nil
}

func (s *cartServiceImpl) AddItemToCart(ctx context.Context, userID uuid.UUID, req *models.CartRequest) ([]models.CartItemResponse, error) {

	product, err := s.productRepo.GetProductByID(ctx, req.ProductID)
	if err != nil {
		if stdErrors.Is(err, errors.ErrCartNotFound) {
			return nil, errors.ErrCartNotFound
		}
		return nil, fmt.Errorf("product not found: %w", err)
	}

	// checking stock
	if product.Stock < req.Quantity {
		return nil, errors.ErrInvalidRequestPayload
	}

	cartID := helpers.GenerateNewID()

	cartData := &entities.Cart{
		ID:          cartID,
		ProductID:   req.ProductID,
		UserID:      userID,
		Quantity:    req.Quantity,
		Description: req.Description,
	}

	err = s.cartRepo.AddItemToCart(ctx, userID, cartID, cartData)
	if err != nil {
		log.Printf("Error in service adding item %s to cart for user %s: %v", cartData.ProductID, userID, err)
		return nil, fmt.Errorf("failed to add item to cart: %w", err)
	}

	createdData, err := s.GetCartItemsByUserID(ctx, userID)
	if err != nil {
		log.Printf("Error in service getting created cart for user %s: %v", userID, err)
	}

	return createdData, nil
}

func (s *cartServiceImpl) RemoveItemFromCart(ctx context.Context, userID, productID uuid.UUID) error {
	err := s.cartRepo.RemoveItemFromCart(ctx, userID, productID)
	if err != nil {
		if stdErrors.Is(err, errors.ErrCartNotFound) {
			return errors.ErrCartNotFound
		}

		log.Printf("Error in service removing item %s from cart for user %s: %v", productID, userID, err)
		return fmt.Errorf("gagal menghapus item dari keranjang: %w", err)
	}
	return nil
}

func (s *cartServiceImpl) UpdateItemQuantity(ctx context.Context, productID, userID uuid.UUID, req *models.UpdateCartRequest) ([]models.CartItemResponse, error) {
	cartData := &entities.Cart{
		Quantity: req.Quantity,
	}

	if req.Description != "" {
		cartData.Description = req.Description
	}

	err := s.cartRepo.UpdateItemQuantity(ctx, productID, userID, cartData)
	if err != nil {
		if stdErrors.Is(err, errors.ErrCartNotFound) {
			return nil, errors.ErrCartNotFound
		}

		log.Printf("Error in service updating item %s quantity for user %s: %v", productID, userID, err)
		return nil, fmt.Errorf("gagal mengupdate kuantitas item: %w", err)
	}

	updatedCart, err := s.GetCartItemsByUserID(ctx, userID)
	if err != nil {
		log.Printf("Error in service getting updated cart for user %s: %v", userID, err)
	}
	return updatedCart, nil
}

func (s *cartServiceImpl) RestoreCartFromDB(ctx context.Context, userID uuid.UUID) ([]models.CartItemResponse, error) {
	err := s.cartRepo.RestoreCartFromDB(ctx, userID)
	if err != nil {
		if stdErrors.Is(err, errors.ErrCartNotFound) {
			return nil, errors.ErrCartNotFound
		}

		log.Printf("Error in service restoring cart for user %s: %v", userID, err)
		return nil, fmt.Errorf("failed to restore cart: %w", err)
	}

	restoredCart, err := s.GetCartItemsByUserID(ctx, userID)
	if err != nil {
		log.Printf("Error in service getting restored cart for user %s: %v", userID, err)
		return nil, fmt.Errorf("failed to get restored cart: %w", err)
	}

	return restoredCart, nil

}

func (s *cartServiceImpl) CheckoutCart(ctx context.Context, userID uuid.UUID) error {
	// Lakukan checkout di repository (yang akan memindahkan data ke DB, mengurangi stok, dll.)
	order, err := s.cartRepo.CheckoutCart(ctx, userID) // Asumsi CheckoutCart repo mengembalikan *models.Order
	if err != nil {
		log.Printf("Error in service processing checkout for user %s: %v", userID, err)
		return fmt.Errorf("gagal memproses checkout: %w", err)
	}

	// --- Publikasikan Event OrderCreated ke RabbitMQ ---
	productIDs := make([]string, 0)
	quantities := make(map[string]int)
	for _, item := range order.OrderItems {
		productIDs = append(productIDs, item.ProductID.String())
		quantities[item.ProductID.String()] = item.Quantity
	}

	event := models.OrderCreatedEvent{
		OrderID:     order.ID.String(),
		UserID:      order.UserID.String(),
		TotalAmount: order.TotalAmount,
		OrderDate:   order.OrderDate,
		ProductIDs:  productIDs,
		Quantities:  quantities,
	}

	eventBytes, err := json.Marshal(event)
	if err != nil {
		log.Printf("Error marshaling OrderCreatedEvent: %v", err)
		// Ini adalah error yang tidak menghentikan checkout, tetapi perlu dicatat
		return fmt.Errorf("checkout successful but failed to publish event: %w", err)
	}

	// Publikasikan pesan ke RabbitMQ
	err = s.rabbitmqClient.PublishMessage(eventBytes)
	if err != nil {
		log.Printf("Error publishing OrderCreatedEvent to RabbitMQ: %v", err)
		// Ini juga error yang tidak menghentikan checkout utama, tapi perlu dicatat/ditangani lebih lanjut
		return fmt.Errorf("checkout successful but failed to publish event: %w", err)
	}

	log.Printf("Order %s successfully checked out and event published to RabbitMQ.", order.ID)
	return nil
}
