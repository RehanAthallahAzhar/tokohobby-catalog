package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"

	"github.com/RehanAthallahAzhar/shopeezy-inventory-cart/db"
	dbGenerated "github.com/RehanAthallahAzhar/shopeezy-inventory-cart/internal/db"
	"github.com/RehanAthallahAzhar/shopeezy-inventory-cart/internal/delivery/http/routes"
	"github.com/RehanAthallahAzhar/shopeezy-inventory-cart/internal/handlers"
	"github.com/RehanAthallahAzhar/shopeezy-inventory-cart/internal/models"
	"github.com/RehanAthallahAzhar/shopeezy-inventory-cart/internal/pkg/authclient"
	"github.com/RehanAthallahAzhar/shopeezy-inventory-cart/internal/pkg/rabbitmq"
	"github.com/RehanAthallahAzhar/shopeezy-inventory-cart/internal/pkg/redisclient"
	"github.com/RehanAthallahAzhar/shopeezy-inventory-cart/internal/repositories"
	"github.com/RehanAthallahAzhar/shopeezy-inventory-cart/internal/services"
)

func main() {
	if err := godotenv.Load(); err != nil {
		panic("Error loading .env file: " + err.Error())
	}

	// DB credential from .env
	port, err := strconv.Atoi(os.Getenv("DB_PORT"))
	if err != nil {
		log.Fatal("Invalid DB_PORT: ", err)
	}

	dbCredential := models.Credential{
		Host:         os.Getenv("DB_HOST"),
		Username:     os.Getenv("DB_USER"),
		Password:     os.Getenv("DB_PASSWORD"),
		DatabaseName: os.Getenv("DB_NAME"),
		Port:         port,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Connect to DB
	conn, err := db.Connect(ctx, &dbCredential)
	if err != nil {
		log.Fatalf("DB connection error: %v", err)
	}
	defer conn.Close()

	// Init SQLC query
	sqlcQueries := dbGenerated.New(conn)

	// setup redis
	redisClient, err := redisclient.NewRedisClient()
	if err != nil {
		log.Fatalf("Gagal menginisialisasi klien Redis: %v", err)
	}
	defer redisClient.Close() // Pastikan koneksi Redis ditutup

	// setup gRPC
	accountGRPCServerAddress := os.Getenv("ACCOUNT_GRPC_SERVER_ADDRESS")
	if accountGRPCServerAddress == "" {
		accountGRPCServerAddress = "localhost:50051"
	}

	authClient, err := authclient.NewAuthClient(accountGRPCServerAddress)
	if err != nil {
		log.Fatalf("Gagal membuat klien gRPC Auth: %v", err)
	}
	defer authClient.Close()

	// setup RabbitMQ
	rabbitmqClient, err := rabbitmq.NewRabbitMQClient("order_created_queue") // Nama queue untuk event order
	if err != nil {
		log.Fatalf("Failed to initialize RabbitMQ client: %v", err)
	}
	defer rabbitmqClient.Close() // Pastikan koneksi RabbitMQ ditutup

	// setup repo
	productsRepo := repositories.NewProductRepository(sqlcQueries, redisClient)
	cartsRepo := repositories.NewCartRepository(sqlcQueries, conn, productsRepo, redisClient)

	// setup service
	validate := validator.New()

	// setup service
	productService := services.NewProductService(productsRepo, validate)
	cartService := services.NewCartService(cartsRepo, productsRepo, rabbitmqClient)

	e := echo.New()

	// Middleware default
	// e.Use(middleware.Logger())
	// e.Use(middleware.Recover())

	handler := handlers.NewHandler(authClient, productsRepo, cartsRepo, productService, cartService)
	routes.InitRoutes(e, handler)

	log.Printf("Server Echo Cashier App mendengarkan di port 1323")
	e.Logger.Fatal(e.Start(":1323"))
}
