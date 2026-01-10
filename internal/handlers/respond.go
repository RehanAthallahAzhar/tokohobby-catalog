package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/models"
	apperrors "github.com/RehanAthallahAzhar/tokohobby-catalog/internal/pkg/errors"
	"github.com/labstack/echo/v4"
)

const (
	MsgProductRetrieved = "Product retrieved successfully"
	MsgProductCreated   = "Product created successfully"
	MsgProductUpdated   = "Product updated successfully"
	MsgProductDeleted   = "Product deleted successfully"

	MsgFailedToRetrieveProduct = "Failed to retrieve product"
	MsgFailedToCreateProduct   = "Failed to create product"
	MsgFailedToUpdateProduct   = "Failed to update product"
	MsgFailedToDeleteProduct   = "Failed to delete product"

	MsgCartRetrieved       = "Cart retrieved successfully"
	MsgCartCreated         = "Cart created successfully"
	MsgCartUpdated         = "Cart updated successfully"
	MsgCartDeleted         = "Cart deleted successfully"
	MsgCartCleared         = "Cart cleared successfully"
	MsgCartCheckedOut      = "Cart checked out successfully"
	MsgFailedToRestoreCart = "Failed to restore cart"

	MsgFailedToAddItemToCart = "Failed to add item to cart"
	MsgFailedToRetrieveCart  = "Failed to retrieve cart"
	MsgFailedToUpdateCart    = "Failed to update cart"
	MsgFailedToDeleteCart    = "Failed to delete cart"

	MsgProductAddedToCart = "Product added to cart successfully"

	MsgNotifyProductAddedToCart = "Product added to cart notification"
	MsgNotifyCartDeleted        = "Cart deleted notification"

	MsgNotifyProductCreated = "Product created notification"
	MsgNotifyProductUpdated = "Product updated notification"
	MsgNotifyProductDeleted = "Product deleted notification"
)

func respondSuccess(c echo.Context, status int, message string, data interface{}) error {
	return c.JSON(status, models.SuccessResponse{
		Message: message,
		Data:    data,
	})
}

func respondError(c echo.Context, status int, err error) error {
	return c.JSON(status, models.ErrorResponse{
		Error: err.Error(),
	})
}

func handleGetError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, apperrors.ErrInvalidUserInput),
		errors.Is(err, apperrors.ErrInvalidCartOperation):
		return respondError(c, http.StatusBadRequest, err)

	case errors.Is(err, apperrors.ErrInsufficientStock),
		errors.Is(err, apperrors.ErrCartAlreadyCheckedOut):
		return respondError(c, http.StatusForbidden, err)

	case errors.Is(err, apperrors.ErrInternalServerError):
		return respondError(c, http.StatusInternalServerError, err)

	default:
		return respondError(c, http.StatusInternalServerError, fmt.Errorf("%w: %s", apperrors.ErrInternalServerError, err))
	}
}

func handleOperationError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, apperrors.ErrProductNotBelongToSeller),
		errors.Is(err, apperrors.ErrInvalidUserInput),
		errors.Is(err, apperrors.ErrInvalidCartOperation):
		return respondError(c, http.StatusForbidden, err)

	case err.Error() == apperrors.ErrInvalidProductUpdatePayload.Error(),
		errors.Is(err, apperrors.ErrInsufficientStock),
		errors.Is(err, apperrors.ErrCartAlreadyCheckedOut):
		return respondError(c, http.StatusBadRequest, apperrors.ErrInvalidRequestPayload)

	default:
		c.Logger().Errorf("%s: %v", apperrors.ErrInternalServerError, err)
		return respondError(c, http.StatusInternalServerError, fmt.Errorf("internal server error"))
	}
}
