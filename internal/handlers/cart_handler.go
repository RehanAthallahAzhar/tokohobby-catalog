package handlers

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"

	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/entities"
	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/models"
	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/pkg/errors"
	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/pkg/messaging"
	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/services"
)

type CartHandler struct {
	CartSvc    services.CartService
	MsgManager *messaging.Manager
	log        *logrus.Logger
}

func NewCartHandler(
	cartSvc services.CartService,
	msgManager *messaging.Manager,
	log *logrus.Logger,
) *CartHandler {
	return &CartHandler{
		CartSvc:    cartSvc,
		MsgManager: msgManager,
		log:        log,
	}
}

func (h *CartHandler) AddToCart() echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		userID, err := getUserIDFromContext(c)
		if err != nil {
			return respondError(c, http.StatusUnauthorized, errors.ErrInvalidUserSession)
		}

		productID, err := getIDFromPathParam(c, "product_id")
		if err != nil {
			return respondError(c, http.StatusBadRequest, err)
		}

		var req models.CartRequest
		if err := c.Bind(&req); err != nil {
			return respondError(c, http.StatusBadRequest, errors.ErrInvalidRequestPayload)
		}

		if err := h.CartSvc.AddItemToCart(ctx, userID, productID, &req); err != nil {
			return handleOperationError(c, err)
		}

		h.MsgManager.Send(messaging.NotificationPayload{
			Type:    MsgNotifyProductAddedToCart,
			UserID:  userID,
			Message: MsgProductAddedToCart,
		})

		return respondSuccess(c, http.StatusOK, MsgCartCreated, nil)
	}
}

func (h *CartHandler) GetCartItemsByUserID() echo.HandlerFunc {
	return func(c echo.Context) error {
		logrus.Info("request GetCartItemsByUserID")
		ctx := c.Request().Context()

		userID, err := getUserIDFromContext(c)
		if err != nil {
			return respondError(c, http.StatusUnauthorized, errors.ErrInvalidUserSession)
		}

		res, err := h.CartSvc.GetCartItemsByUserID(ctx, userID)
		if err != nil {
			return handleGetError(c, err)
		}

		return respondSuccess(c, http.StatusOK, MsgCartRetrieved, toCartResponse(res))
	}
}

func (h *CartHandler) UpdateCartItem() echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		userID, err := getUserIDFromContext(c)
		if err != nil {
			return respondError(c, http.StatusUnauthorized, errors.ErrInvalidUserSession)
		}

		productIDStr := c.Param("product_id")
		productID, err := uuid.Parse(productIDStr)
		if err != nil {
			return respondError(c, http.StatusBadRequest, errors.ErrInvalidRequestPayload)
		}

		var req models.UpdateCartRequest
		if err := c.Bind(&req); err != nil {
			return respondError(c, http.StatusBadRequest, errors.ErrInvalidRequestPayload)
		}

		logger := h.log.WithFields(logrus.Fields{"user_id": userID, "product_id": productID, "new_quantity": req.Quantity})
		logger.Info("Receiving UpdateCartItem requests")

		err = h.CartSvc.UpdateItem(ctx, userID, productID, req.Quantity, req.Description)
		if err != nil {
			logger.WithError(err).Error("Error dari service saat memperbarui item keranjang")
			return handleOperationError(c, err)
		}

		return respondSuccess(c, http.StatusOK, MsgCartUpdated, nil)
	}
}

func (h *CartHandler) RemoveFromCart() echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		userID, err := getUserIDFromContext(c)
		if err != nil {
			return respondError(c, http.StatusUnauthorized, errors.ErrInvalidUserSession)
		}

		productID, err := getIDFromPathParam(c, "product_id")
		if err != nil {
			return respondError(c, http.StatusBadRequest, err)
		}

		err = h.CartSvc.RemoveItemFromCart(ctx, userID, productID)
		if err != nil {
			return handleOperationError(c, err)
		}

		h.MsgManager.Send(messaging.NotificationPayload{
			Type:    MsgNotifyCartDeleted,
			UserID:  userID,
			Message: MsgCartDeleted,
		})

		return respondSuccess(c, http.StatusOK, MsgCartDeleted, nil)
	}
}

// ------- HELPERS -------

func toCartResponse(cart *entities.Cart) *models.CartResponse {
	return &models.CartResponse{
		UserID:     cart.UserID.String(),
		TotalItems: cart.TotalItems,
		Items:      toCartItemsResponse(cart.Items),
	}
}

func toCartItemsResponse(items []entities.CartItem) []models.CartItemResponse {
	cartItemsResponse := make([]models.CartItemResponse, len(items))

	for i, item := range items {
		cartItemsResponse[i] = *toCartItemResponse(item)
	}

	return cartItemsResponse
}

func toCartItemResponse(item entities.CartItem) *models.CartItemResponse {
	return &models.CartItemResponse{
		SellerName:   item.SellerName,
		ProductID:    item.ProductID.String(),
		ProductName:  item.ProductName,
		ProductImage: "",
		Price:        item.Price,
		Quantity:     item.Quantity,
		Description:  item.Description,
		Checked:      item.Checked,
	}
}
