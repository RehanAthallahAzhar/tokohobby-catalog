package routes

import (
	"github.com/RehanAthallahAzhar/shopeezy-inventory-cart/internal/delivery/http/middlewares"
	"github.com/RehanAthallahAzhar/shopeezy-inventory-cart/internal/handlers"

	"github.com/labstack/echo/v4"
)

func InitRoutes(e *echo.Echo, handler *handlers.API) {
	// Route group that requires gRPC authentication
	authGroup := e.Group("/api/v1")
	authGroup.Use(handler.AuthMiddleware) // Apply authentication middleware

	productGroup := authGroup.Group("/products")
	productGroup.GET("/", handler.GetAllProducts())
	productGroup.GET("/tag/:tag", handler.GetProductsByName())
	productGroup.GET("/:id", handler.GetProductByID(), middlewares.RequireRoles("admin"))
	productGroup.GET("/seller/:seller_id", handler.GetProductsBySellerID())
	productGroup.POST("/create", handler.CreateProduct, middlewares.RequireRoles("admin", "seller"))
	productGroup.PUT("/update/:id", handler.UpdateProduct, middlewares.RequireRoles("admin", "seller"))
	productGroup.DELETE("/delete/:id", handler.DeleteProduct, middlewares.RequireRoles("admin", "seller"))

	cartGroup := authGroup.Group("/cart")
	cartGroup.GET("/", handler.GetCartItemsByUserID())
	cartGroup.GET("/product/:product_id", handler.GetCartItemByProductID())
	cartGroup.POST("/add", handler.AddToCart)
	cartGroup.PUT("/update/:product_id", handler.UpdateCartItemQuantity)
	cartGroup.DELETE("/remove/:product_id", handler.RemoveFromCart)
	cartGroup.POST("/restore", handler.AddToCart)
	cartGroup.GET("/checkout", handler.CheckoutCart)
}
