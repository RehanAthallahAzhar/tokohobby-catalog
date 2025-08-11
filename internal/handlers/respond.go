package handlers

import (
	stdErrors "errors"
	"fmt"
	"log"
	"net/http"

	"github.com/RehanAthallahAzhar/shopeezy-inventory-cart/internal/models"
	"github.com/RehanAthallahAzhar/shopeezy-inventory-cart/internal/pkg/errors"
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
	case stdErrors.Is(err, errors.ErrProductNotFound),
		stdErrors.Is(err, errors.ErrCartNotFound):
		return respondError(c, http.StatusNotFound, err)

	case stdErrors.Is(err, errors.ErrInvalidUserInput),
		stdErrors.Is(err, errors.ErrInvalidCartOperation): // misalnya operasi cart yang tak sesuai
		return respondError(c, http.StatusBadRequest, err)

	case stdErrors.Is(err, errors.ErrInsufficientStock),
		stdErrors.Is(err, errors.ErrCartAlreadyCheckedOut):
		return respondError(c, http.StatusForbidden, err)

	case stdErrors.Is(err, errors.ErrInternalServerError):
		return respondError(c, http.StatusInternalServerError, err)

	default:
		log.Printf("Unhandled inventory error: %v", err)
		return respondError(c, http.StatusInternalServerError, fmt.Errorf("%w: %s", errors.ErrInternalServerError, err))
	}
}

func handleOperationError(c echo.Context, err error, contextMsg string) error {
	switch {
	case stdErrors.Is(err, errors.ErrProductNotFound),
		stdErrors.Is(err, errors.ErrCartNotFound):
		return respondError(c, http.StatusNotFound, err)

	case stdErrors.Is(err, errors.ErrProductNotBelongToSeller),
		stdErrors.Is(err, errors.ErrInvalidUserInput),
		stdErrors.Is(err, errors.ErrInvalidCartOperation):
		return respondError(c, http.StatusForbidden, err)

	case err.Error() == errors.ErrInvalidProductUpdatePayload.Error(),
		stdErrors.Is(err, errors.ErrInsufficientStock),
		stdErrors.Is(err, errors.ErrCartAlreadyCheckedOut):
		return respondError(c, http.StatusBadRequest, errors.ErrInvalidRequestPayload)

	default:
		c.Logger().Errorf("%s: %v", contextMsg, err)
		return respondError(c, http.StatusInternalServerError, fmt.Errorf("internal server error"))
	}
}
