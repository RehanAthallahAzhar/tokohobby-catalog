package handlers

import (
	"log"
	"net/http"
	"strings"

	"github.com/RehanAthallahAzhar/shopeezy-inventory-cart/internal/helpers"
	"github.com/RehanAthallahAzhar/shopeezy-inventory-cart/internal/pkg/authclient"
	"github.com/RehanAthallahAzhar/shopeezy-inventory-cart/internal/pkg/errors"
	"github.com/RehanAthallahAzhar/shopeezy-inventory-cart/internal/repositories"
	"github.com/RehanAthallahAzhar/shopeezy-inventory-cart/internal/services"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// API struct untuk mengelola handler dan dependensinya.
type API struct {
	AuthGRPCClient *authclient.AuthClient
	ProductRepo    repositories.ProductRepository
	CartRepo       repositories.CartRepository
	ProductSvc     services.ProductService
	CartSvc        services.CartService
}

// NewHandler membuat instance baru dari API.
func NewHandler(
	authGRPCClient *authclient.AuthClient,
	productRepo repositories.ProductRepository,
	cartRepo repositories.CartRepository,
	productSvc services.ProductService,
	cartSvc services.CartService,
) *API {
	return &API{
		AuthGRPCClient: authGRPCClient,
		ProductRepo:    productRepo,
		CartRepo:       cartRepo,
		ProductSvc:     productSvc,
		CartSvc:        cartSvc,
	}
}

func (a *API) AuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		authHeader := c.Request().Header.Get("Authorization")
		if authHeader == "" {
			return c.JSON(http.StatusUnauthorized, echo.Map{"message": "Authorization token not found"})
		}

		token := authHeader
		log.Println("Extracted token:", token)
		if len(authHeader) > 7 && strings.HasPrefix(authHeader, "Bearer ") {
			token = authHeader[7:]
		} else {
			return c.JSON(http.StatusBadRequest, echo.Map{"message": "Invalid token format (expected Bearer token)"})
		}

		isValid, userID, username, role, errMsg, err := a.AuthGRPCClient.ValidateToken(token)
		if err != nil {
			log.Printf("Error during gRPC token validation: %v", err)
			return c.JSON(http.StatusInternalServerError, echo.Map{"message": "Server error during token validation"})
		}

		if !isValid {
			return c.JSON(http.StatusUnauthorized, echo.Map{"message": errMsg})
		}

		c.Set("userID", userID)
		c.Set("username", username)
		c.Set("role", role)
		log.Printf("User %s (ID: %s, Role: %s) successfully authenticated.", username, userID, role)

		return next(c)
	}
}

func extractUserID(c echo.Context) (uuid.UUID, error) {
	if val := c.Get("userID"); val != nil {
		if id, ok := val.(string); ok {

			res, err := helpers.StringToUUID(id)
			if err != nil {
				return uuid.Nil, err
			}

			return res, nil

		}
	}

	return uuid.Nil, errors.ErrInvalidUserSession
}
