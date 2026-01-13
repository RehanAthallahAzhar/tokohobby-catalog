package routes

import (
	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/delivery/http/middlewares"
	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/handlers"

	"github.com/labstack/echo/v4"
)

func InitRoutes(e *echo.Echo, productHandler *handlers.ProductHandler, cartHandler *handlers.CartHandler, authMiddleware echo.MiddlewareFunc) {

	api := e.Group("/api")

	productPublic := api.Group("/products")
	{
		productPublic.GET("/", productHandler.GetAllProducts())
		productPublic.GET("/name/:name", productHandler.GetProductsByName())
		productPublic.GET("/category/:type", productHandler.GetProductsByType())
		productPublic.GET("/:id", productHandler.GetProductByID())
		productPublic.GET("/seller/:seller_id", productHandler.GetProductsBySellerID())
	}

	protectedApi := api
	protectedApi.Use(authMiddleware)

	productProtected := protectedApi.Group("/products")
	{
		productProtected.POST("/", productHandler.CreateProduct(), middlewares.RequireRoles("admin", "seller"))
		productProtected.PUT("/:product_id", productHandler.UpdateProduct(), middlewares.RequireRoles("admin", "seller"))
		productProtected.DELETE("/:product_id", productHandler.DeleteProduct(), middlewares.RequireRoles("admin", "seller"))
		productProtected.DELETE("/clear-cache", productHandler.ClearProductCaches())
	}

	cart := protectedApi.Group("/cart")
	{
		cart.GET("/", cartHandler.GetCartItemsByUserID())
		cart.POST("/:product_id", cartHandler.AddToCart())
		cart.PUT("/:product_id", cartHandler.UpdateCartItem())
		cart.DELETE("/:product_id", cartHandler.RemoveFromCart())
	}
}
