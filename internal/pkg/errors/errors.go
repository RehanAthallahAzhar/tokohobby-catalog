package errors

import "errors"

var (
	// validate
	ErrInvalidRequestPayload = errors.New("invalid request payload")

	//status
	ErrInternalServerError = errors.New("internal server error")

	// product
	ErrProductNotFound             = errors.New("product not found")
	ErrInsufficientStock           = errors.New("insufficient stock for this quantity")
	ErrInvalidUserInput            = errors.New("invalid user input")
	ErrProductNotBelongToSeller    = errors.New("product does not belong to this seller")
	ErrInvalidProductUpdatePayload = errors.New("all required columns must not be empty and valid for update")

	//cart
	ErrCartNotFound          = errors.New("cart item not found")
	ErrInvalidCartOperation  = errors.New("invalid cart operation")
	ErrCartAlreadyCheckedOut = errors.New("cart is already checked out")
	ErrCartRetrievalFail     = errors.New("failed to retrieve cart")
	ErrCartEmpty             = errors.New("cart is empty")

	//cart-item
	ErrCartItemNotFound = errors.New("cart item not found")

	//order
	ErrOrderNotFound = errors.New("order not found")

	//auth
	ErrInvalidUserSession = errors.New("invalid user session")
	ErrUnauthorized       = errors.New("unauthorized")
)

// var ErrNotFound = errors.New("not found")
// var ErrUnauthorized = errors.New("unauthorized")
// var ErrForbidden = errors.New("forbidden")
// var ErrConflict = errors.New("conflict")
// var ErrBadRequest = errors.New("bad request")
// var ErrNotImplemented = errors.New("not implemented")
// var ErrServiceUnavailable = errors.New("service unavailable")
// var ErrGatewayTimeout = errors.New("gateway timeout")
// var ErrGatewayClosed = errors.New("gateway closed")
