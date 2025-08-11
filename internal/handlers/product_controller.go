package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/RehanAthallahAzhar/shopeezy-inventory-cart/internal/helpers"
	"github.com/RehanAthallahAzhar/shopeezy-inventory-cart/internal/models"
	"github.com/RehanAthallahAzhar/shopeezy-inventory-cart/internal/pkg/errors"
)

func (api *API) GetAllProducts() echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		c.Logger().Infof("Received request for product list from IP: %s", c.RealIP())

		res, err := api.ProductSvc.GetAllProducts(ctx)
		if err != nil {
			return handleGetError(c, err)
		}

		return respondSuccess(c, http.StatusOK, MsgProductRetrieved, res)
	}
}

func (api *API) GetProductsByName() echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		ProductName, err := helpers.GetFromPathParam(c, "tag")
		if err != nil {
			return respondError(c, http.StatusBadRequest, err)
		}

		res, err := api.ProductSvc.GetProductsByName(ctx, ProductName)
		if err != nil {
			return handleGetError(c, err)
		}

		return respondSuccess(c, http.StatusOK, MsgProductRetrieved, res)
	}
}

func (api *API) GetProductByID() echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		c.Logger().Infof("Received request for GetProductByID from IP: %s", c.RealIP())

		productID, err := helpers.GetIDFromPathParam(c, "id")
		if err != nil {
			return respondError(c, http.StatusBadRequest, err)
		}

		res, err := api.ProductSvc.GetProductByID(ctx, productID)
		if err != nil {
			return handleGetError(c, err)
		}

		return respondSuccess(c, http.StatusOK, MsgProductRetrieved, res)
	}
}

func (api *API) GetProductsBySellerID() echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		sellerID, err := helpers.GetIDFromPathParam(c, "seller_id")
		if err != nil {
			return respondError(c, http.StatusBadRequest, err)
		}

		res, err := api.ProductSvc.GetProductsBySellerID(ctx, sellerID)
		if err != nil {
			return handleGetError(c, err)
		}

		return respondSuccess(c, http.StatusOK, MsgProductRetrieved, res)
	}
}

func (api *API) CreateProduct(c echo.Context) error {
	ctx := c.Request().Context()

	userID, err := extractUserID(c)
	if err != nil {
		return respondError(c, http.StatusUnauthorized, errors.ErrInvalidUserSession)
	}

	var req models.ProductRequest
	if err := c.Bind(&req); err != nil {
		return respondError(c, http.StatusBadRequest, errors.ErrInvalidRequestPayload)
	}

	res, err := api.ProductSvc.CreateProduct(ctx, userID, &req)
	if err != nil {
		return handleOperationError(c, err, MsgFailedToCreateProduct)
	}

	return respondSuccess(c, http.StatusCreated, MsgProductCreated, res)
}

func (api *API) UpdateProduct(c echo.Context) error {
	ctx := c.Request().Context()

	userID, err := extractUserID(c)
	if err != nil {
		return respondError(c, http.StatusUnauthorized, errors.ErrInvalidUserSession)
	}

	productID, err := helpers.GetIDFromPathParam(c, "id")
	if err != nil {
		return respondError(c, http.StatusBadRequest, err)
	}

	var productData models.ProductRequest
	if err := c.Bind(&productData); err != nil {
		return respondError(c, http.StatusBadRequest, errors.ErrInvalidRequestPayload)
	}

	res, err := api.ProductSvc.UpdateProduct(ctx, &productData, userID, productID)
	if err != nil {
		return handleOperationError(c, err, MsgFailedToUpdateProduct)
	}

	return respondSuccess(c, http.StatusOK, MsgProductUpdated, res)
}

func (api *API) DeleteProduct(c echo.Context) error {
	ctx := c.Request().Context()

	sellerID, err := extractUserID(c)
	if err != nil {
		return respondError(c, http.StatusUnauthorized, errors.ErrInvalidUserSession)
	}

	productID, err := helpers.GetIDFromPathParam(c, "id")
	if err != nil {
		return respondError(c, http.StatusBadRequest, err)
	}

	err = api.ProductSvc.DeleteProduct(ctx, productID, sellerID)
	if err != nil {
		return handleOperationError(c, err, MsgFailedToDeleteProduct)
	}

	return respondSuccess(c, http.StatusOK, MsgProductDeleted, nil)
}
