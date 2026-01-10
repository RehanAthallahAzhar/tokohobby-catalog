package routes

import (
	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/delivery/http/middlewares"
	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/handlers"

	"github.com/labstack/echo/v4"
)

func InitRoutes(e *echo.Echo, productHandler *handlers.ProductHandler, cartHandler *handlers.CartHandler, authMiddleware echo.MiddlewareFunc) {

	publicGroup := e.Group("/api/v1")

	productPublicGroup := publicGroup.Group("/products")
	{
		productPublicGroup.GET("/", productHandler.GetAllProducts())
		productPublicGroup.GET("/name/:name", productHandler.GetProductsByName())
		productPublicGroup.GET("/category/:type", productHandler.GetProductsByType())
		productPublicGroup.GET("/:id", productHandler.GetProductByID())
		productPublicGroup.GET("/seller/:seller_id", productHandler.GetProductsBySellerID())
	}

	authGroup := e.Group("/api/v1")
	authGroup.Use(authMiddleware)

	productAuthGroup := authGroup.Group("/products")
	{
		productAuthGroup.POST("/create", productHandler.CreateProduct(), middlewares.RequireRoles("admin", "seller"))
		productAuthGroup.PUT("/update/:product_id", productHandler.UpdateProduct(), middlewares.RequireRoles("admin", "seller"))
		productAuthGroup.DELETE("/delete/:product_id", productHandler.DeleteProduct(), middlewares.RequireRoles("admin", "seller"))
		productAuthGroup.DELETE("/clear-cache", productHandler.ClearProductCaches(), middlewares.RequireRoles("admin")) // Reset cache harus diproteksi
	}

	cartGroup := authGroup.Group("/cart")
	{
		cartGroup.GET("/", cartHandler.GetCartItemsByUserID())
		cartGroup.POST("/add/:product_id", cartHandler.AddToCart())
		cartGroup.PUT("/update/:product_id", cartHandler.UpdateCartItem())
		cartGroup.DELETE("/remove/:product_id", cartHandler.RemoveFromCart())
	}
}
