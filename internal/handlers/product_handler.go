package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"

	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/entities"
	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/helpers"
	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/models"
	apperrors "github.com/RehanAthallahAzhar/tokohobby-catalog/internal/pkg/errors"
	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/services"
)

type ProductHandler struct {
	ProductSvc services.ProductService
	log        *logrus.Logger
}

func NewProductHandler(
	productSvc services.ProductService,
	log *logrus.Logger,
) *ProductHandler {
	return &ProductHandler{
		ProductSvc: productSvc,
		log:        log,
	}
}

func (p *ProductHandler) CreateProduct() echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		userID, err := getUserIDFromContext(c)
		if err != nil {
			return respondError(c, http.StatusUnauthorized, apperrors.ErrInvalidUserSession)
		}

		var req models.ProductRequest
		if err := c.Bind(&req); err != nil {
			return respondError(c, http.StatusBadRequest, apperrors.ErrInvalidRequestPayload)
		}

		res, err := p.ProductSvc.CreateProduct(ctx, userID, &req)
		if err != nil {
			return handleOperationError(c, err)
		}

		return respondSuccess(c, http.StatusCreated, MsgProductCreated, toProductResponse(res))
	}
}

func (p *ProductHandler) GetAllProducts() echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		res, err := p.ProductSvc.GetAllProducts(ctx)
		if err != nil {
			return handleGetError(c, err)
		}

		return respondSuccess(c, http.StatusOK, MsgProductRetrieved, toProductResponseList(res))
	}
}

func (p *ProductHandler) GetProductsByName() echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		productName, err := getFromPathParam(c, "name")
		if err != nil {
			return respondError(c, http.StatusBadRequest, err)
		}

		res, err := p.ProductSvc.GetProductsByName(ctx, productName)
		if err != nil {
			return handleGetError(c, err)
		}

		return respondSuccess(c, http.StatusOK, MsgProductRetrieved, toProductResponseList(res))
	}
}

func (p *ProductHandler) GetProductsByType() echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		productType, err := getFromPathParam(c, "type")
		if err != nil {
			return respondError(c, http.StatusBadRequest, err)
		}

		res, err := p.ProductSvc.GetProductsByType(ctx, productType)
		if err != nil {
			return handleGetError(c, err)
		}

		return respondSuccess(c, http.StatusOK, MsgProductRetrieved, toProductResponseList(res))
	}
}

func (p *ProductHandler) GetProductByID() echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		productID, err := getIDFromPathParam(c, "id")
		if err != nil {
			return respondError(c, http.StatusBadRequest, err)
		}

		res, err := p.ProductSvc.GetProductByID(ctx, productID)
		if err != nil {
			return handleGetError(c, err)
		}

		return respondSuccess(c, http.StatusOK, MsgProductRetrieved, toProductResponse(res))
	}
}

func (p *ProductHandler) GetProductsBySellerID() echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		sellerID, err := getIDFromPathParam(c, "seller_id")
		if err != nil {
			return respondError(c, http.StatusBadRequest, err)
		}

		res, err := p.ProductSvc.GetProductsBySellerID(ctx, sellerID)
		if err != nil {
			return handleGetError(c, err)
		}

		return respondSuccess(c, http.StatusOK, MsgProductRetrieved, toProductResponseList(res))
	}
}

func (p *ProductHandler) UpdateProduct() echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		userID, err := getUserIDFromContext(c)
		if err != nil {
			return respondError(c, http.StatusUnauthorized, apperrors.ErrInvalidUserSession)
		}

		role, err := getRoleFromContext(c)
		if err != nil {
			return respondError(c, http.StatusUnauthorized, apperrors.ErrInvalidUserSession)
		}

		productID, err := getIDFromPathParam(c, "product_id")
		if err != nil {
			return respondError(c, http.StatusBadRequest, err)
		}

		var productData models.ProductRequest
		if err := c.Bind(&productData); err != nil {
			return respondError(c, http.StatusBadRequest, apperrors.ErrInvalidRequestPayload)
		}

		res, err := p.ProductSvc.UpdateProduct(ctx, &productData, productID, userID, role)
		if err != nil {
			return handleOperationError(c, err)
		}

		return respondSuccess(c, http.StatusOK, MsgProductUpdated, toProductResponse(res))

	}
}

func (p *ProductHandler) DeleteProduct() echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		sellerID, err := getUserIDFromContext(c)
		if err != nil {
			return respondError(c, http.StatusUnauthorized, apperrors.ErrInvalidUserSession)
		}

		role, err := getRoleFromContext(c)
		if err != nil {
			return respondError(c, http.StatusUnauthorized, apperrors.ErrInvalidUserSession)
		}

		productID, err := getIDFromPathParam(c, "product_id")
		if err != nil {
			return respondError(c, http.StatusBadRequest, err)
		}

		res, err := p.ProductSvc.DeleteProduct(ctx, productID, sellerID, role)
		if err != nil {
			return handleOperationError(c, err)
		}

		return respondSuccess(c, http.StatusOK, MsgProductDeleted, toProductResponse(res))
	}
}

func (p *ProductHandler) ClearProductCaches() echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		err := p.ProductSvc.ResetAllProductCaches(ctx)
		if err != nil {
			return handleOperationError(c, err)
		}

		return respondSuccess(c, http.StatusOK, apperrors.MsgProductCacheCleared, nil)
	}
}

// ------- HELPERS -------
func toProductResponse(product *entities.Product) *models.ProductResponse {
	return &models.ProductResponse{
		ID:          product.ID,
		SellerID:    product.SellerID,
		Name:        product.Name,
		Price:       product.Price,
		Stock:       product.Stock,
		Discount:    product.Discount,
		Type:        product.Type,
		Description: product.Description,
		CreatedAt:   product.CreatedAt.Format(helpers.LAYOUTFORMAT),
		UpdatedAt:   product.UpdatedAt.Format(helpers.LAYOUTFORMAT),
	}
}

func toProductResponseList(products []entities.Product) []*models.ProductResponse {
	var productResponses []*models.ProductResponse

	for _, product := range products {
		productResponses = append(productResponses, toProductResponse(&product))
	}

	return productResponses
}
