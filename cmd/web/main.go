package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/golang-migrate/migrate/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"

	"github.com/RehanAthallahAzhar/tokohobby-catalog/db"
	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/configs"
	customMiddleware "github.com/RehanAthallahAzhar/tokohobby-catalog/internal/delivery/http/middlewares"
	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/delivery/http/routes"
	grpcServerImpl "github.com/RehanAthallahAzhar/tokohobby-catalog/internal/grpc"
	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/handlers"
	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/models"
	dbGenerated "github.com/RehanAthallahAzhar/tokohobby-catalog/internal/pkg/db"
	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/pkg/grpc/account"
	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/pkg/logger"
	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/pkg/redis"
	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/repositories"
	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/services"

	productpb "github.com/RehanAthallahAzhar/tokohobby-protos/pb/product"
)

func main() {
	log := logger.NewLogger()
	log.Println("newlogger executed")

	cfg, err := configs.LoadConfig(log)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	dbCredential := models.Credential{
		Host:         cfg.Database.Host,
		Username:     cfg.Database.User,
		Password:     cfg.Database.Password,
		DatabaseName: cfg.Database.Name,
		Port:         cfg.Database.Port,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	connectionString := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		dbCredential.Username,
		dbCredential.Password,
		dbCredential.Host,
		dbCredential.Port,
		dbCredential.DatabaseName,
	)

	conn, err := db.Connect(ctx, &dbCredential)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}

	// Migrations
	m, err := migrate.New(
		cfg.Migration.Path,
		connectionString,
	)
	if err != nil {
		log.Fatalf("Failed to create migrate instance: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	defer conn.Close()

	// Init SQLC query
	sqlcQueries := dbGenerated.New(conn)

	// Redis
	redisClient, err := redis.NewRedisClient(&cfg.Redis, log)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	accountClientGateway, err := account.NewAccountClient(cfg.GRPC.AccountServiceAddress)
	if err != nil {
		log.Fatalf("Failed to create account client: %v", err)
	}
	defer accountClientGateway.Close()

	authClientGateway, err := account.NewAuthClient(cfg.GRPC.AccountServiceAddress)
	if err != nil {
		log.Fatalf("Failed to create auth client: %v", err)
	}
	defer authClientGateway.Close()

	productsRepo := repositories.NewProductRepository(conn, sqlcQueries, log)
	cartsRepo := repositories.NewCartRepository(redisClient, log)
	validate := validator.New()

	productService := services.NewProductService(productsRepo, redisClient, validate, log)
	cartService := services.NewCartService(cartsRepo, productService, redisClient, accountClientGateway, log)

	productHandler := handlers.NewProductHandler(productService, log)
	cartHandler := handlers.NewCartHandler(cartService, log)

	authMiddleware := customMiddleware.AuthMiddleware(authClientGateway, cfg.Server.JWTSecret, cfg.Server.Audience, log)

	lis, err := net.Listen("tcp", ":"+cfg.Server.GRPCPort)
	if err != nil {
		log.Fatalf("Failed to listen for gRPC server: %v", err)
	}
	s := grpc.NewServer()

	productServer := grpcServerImpl.NewProductServer(productService)
	productpb.RegisterProductServiceServer(s, productServer)
	reflection.Register(s)

	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC server: %v", err)
		}
	}()

	// Setup Echo (REST API)
	e := echo.New()
	e.Use(middleware.RequestID())
	e.Use(customMiddleware.LoggingMiddleware(log))
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"}, // Nginx will handle stricter CORS
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
	}))

	routes.InitRoutes(e, productHandler, cartHandler, authMiddleware)

	e.Logger.Fatal(e.Start(":" + cfg.Server.Port))
}
