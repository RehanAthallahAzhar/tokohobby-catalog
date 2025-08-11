package handlers

import (
	"log"
	"net/http"

	"github.com/RehanAthallahAzhar/shopeezy-inventory-cart/internal/helpers"
	"github.com/RehanAthallahAzhar/shopeezy-inventory-cart/internal/models"
	"github.com/RehanAthallahAzhar/shopeezy-inventory-cart/internal/pkg/errors"
	"github.com/labstack/echo/v4"
)

func (a *API) GetCartItemsByUserID() echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		userID, err := extractUserID(c)
		if err != nil {
			return respondError(c, http.StatusUnauthorized, errors.ErrInvalidUserSession)
		}

		res, err := a.CartSvc.GetCartItemsByUserID(ctx, userID)
		if err != nil {
			return handleGetError(c, err)
		}

		return respondSuccess(c, http.StatusOK, MsgCartRetrieved, res)
	}
}

func (a *API) GetCartItemByProductID() echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		userID, err := extractUserID(c)
		if err != nil {
			return respondError(c, http.StatusUnauthorized, errors.ErrInvalidUserSession)
		}

		productID, err := helpers.GetIDFromPathParam(c, "product_id")
		if err != nil {
			return respondError(c, http.StatusBadRequest, err)
		}

		res, err := a.CartSvc.GetCartItemByProductID(ctx, userID, productID)
		if err != nil {
			return handleGetError(c, err)
		}

		return respondSuccess(c, http.StatusOK, MsgCartRetrieved, res)
	}
}

func (a *API) AddToCart(c echo.Context) error {
	ctx := c.Request().Context()

	userID, err := extractUserID(c)
	if err != nil {
		return respondError(c, http.StatusUnauthorized, errors.ErrInvalidUserSession)
	}

	var req models.CartRequest
	if err := c.Bind(&req); err != nil {
		return respondError(c, http.StatusBadRequest, errors.ErrInvalidRequestPayload)
	}

	res, err := a.CartSvc.AddItemToCart(ctx, userID, &req)
	if err != nil {
		return handleOperationError(c, err, MsgFailedToAddItemToCart)
	}

	return respondSuccess(c, http.StatusOK, MsgCartCreated, res)
}

func (a *API) UpdateCartItemQuantity(c echo.Context) error {
	ctx := c.Request().Context()

	userID, err := extractUserID(c)
	if err != nil {
		return respondError(c, http.StatusUnauthorized, errors.ErrInvalidUserSession)
	}

	productID, err := helpers.GetIDFromPathParam(c, "product_id")
	if err != nil {
		return respondError(c, http.StatusBadRequest, err)
	}

	var req models.UpdateCartRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errors.ErrInvalidRequestPayload)
	}

	res, err := a.CartSvc.UpdateItemQuantity(ctx, userID, productID, &req)
	if err != nil {
		return handleOperationError(c, err, MsgFailedToUpdateCart)
	}

	return respondSuccess(c, http.StatusOK, MsgCartUpdated, res)
}

func (a *API) RemoveFromCart(c echo.Context) error {
	ctx := c.Request().Context()

	userID, err := extractUserID(c)
	if err != nil {
		return respondError(c, http.StatusUnauthorized, errors.ErrInvalidUserSession)
	}

	productID, err := helpers.GetIDFromPathParam(c, "product_id")
	if err != nil {
		return respondError(c, http.StatusBadRequest, err)
	}

	err = a.CartSvc.RemoveItemFromCart(ctx, userID, productID)
	if err != nil {
		return handleOperationError(c, err, MsgFailedToUpdateCart)
	}

	return respondSuccess(c, http.StatusOK, MsgCartDeleted, nil)
}

func (a *API) RestoreCart(c echo.Context) error {
	ctx := c.Request().Context()

	userID, err := extractUserID(c)
	if err != nil {
		return respondError(c, http.StatusUnauthorized, errors.ErrInvalidUserSession)
	}

	res, err := a.CartSvc.RestoreCartFromDB(ctx, userID)
	if err != nil {
		return handleOperationError(c, err, MsgFailedToRestoreCart)
	}

	return respondSuccess(c, http.StatusOK, "Cart restored successfully", res)
}

func (a *API) CheckoutCart(c echo.Context) error {
	ctx := c.Request().Context()

	userID, err := extractUserID(c)
	if err != nil {
		return respondError(c, http.StatusUnauthorized, errors.ErrInvalidUserSession)
	}

	err = a.CartSvc.CheckoutCart(ctx, userID)
	if err != nil {
		if err == errors.ErrCartNotFound {
			return respondError(c, http.StatusNotFound, errors.ErrCartNotFound)
		}

		log.Printf("Error in CheckoutCart handler: %v", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "Failed to checkout cart"})
	}

	return respondSuccess(c, http.StatusOK, MsgCartCheckedOut, nil)
}
