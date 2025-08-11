package middlewares

import (
	"net/http"

	"github.com/RehanAthallahAzhar/shopeezy-inventory-cart/internal/models"

	"github.com/labstack/echo/v4"
)

func RequireRoles(allowedRoles ...string) echo.MiddlewareFunc {
	roleSet := make(map[string]struct{})
	for _, r := range allowedRoles {
		roleSet[r] = struct{}{}
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			role, ok := c.Get("role").(string)
			if !ok {
				return c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "Unauthorized"})
			}

			if _, allowed := roleSet[role]; !allowed {
				return c.JSON(http.StatusForbidden, models.ErrorResponse{Error: "Access denied"})
			}

			return next(c)
		}
	}
}
