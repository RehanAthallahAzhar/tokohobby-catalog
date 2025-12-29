package routes

import (
	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/delivery/http/middlewares"
	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/handlers"

	"github.com/labstack/echo/v4"
)

func InitRoutes(e *echo.Echo, handler *handlers.API, authMiddleware echo.MiddlewareFunc) {

	publicGroup := e.Group("/api/v1")

	productPublicGroup := publicGroup.Group("/products")
	{
		productPublicGroup.GET("/", handler.GetAllProducts())
		productPublicGroup.GET("/name/:name", handler.GetProductsByName())
		productPublicGroup.GET("/category/:type", handler.GetProductsByType())
		productPublicGroup.GET("/:id", handler.GetProductByID())
		productPublicGroup.GET("/seller/:seller_id", handler.GetProductsBySellerID())
	}

	authGroup := e.Group("/api/v1")
	authGroup.Use(authMiddleware)

	productAuthGroup := authGroup.Group("/products")
	{
		productAuthGroup.POST("/create", handler.CreateProduct(), middlewares.RequireRoles("admin", "seller"))
		productAuthGroup.PUT("/update/:product_id", handler.UpdateProduct(), middlewares.RequireRoles("admin", "seller"))
		productAuthGroup.DELETE("/delete/:product_id", handler.DeleteProduct(), middlewares.RequireRoles("admin", "seller"))
		productAuthGroup.DELETE("/clear-cache", handler.ClearProductCaches(), middlewares.RequireRoles("admin")) // Reset cache harus diproteksi
	}

	cartGroup := authGroup.Group("/cart")
	{
		cartGroup.GET("/", handler.GetCartItemsByUserID())
		cartGroup.POST("/add/:product_id", handler.AddToCart())
		cartGroup.PUT("/update/:product_id", handler.UpdateCartItem())
		cartGroup.DELETE("/remove/:product_id", handler.RemoveFromCart())
	}
}
