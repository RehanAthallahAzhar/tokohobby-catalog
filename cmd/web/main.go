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
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"

	"github.com/RehanAthallahAzhar/tokohobby-catalog/db"
	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/configs"
	dbGenerated "github.com/RehanAthallahAzhar/tokohobby-catalog/internal/db"
	customMiddleware "github.com/RehanAthallahAzhar/tokohobby-catalog/internal/delivery/http/middlewares"
	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/delivery/http/routes"
	grpcServerImpl "github.com/RehanAthallahAzhar/tokohobby-catalog/internal/grpc"
	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/handlers"
	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/models"
	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/pkg/grpc/account"
	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/pkg/logger"
	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/pkg/redis"
	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/repositories"
	"github.com/RehanAthallahAzhar/tokohobby-catalog/internal/services"

	accountpb "github.com/RehanAthallahAzhar/tokohobby-protos/pb/account"
	authpb "github.com/RehanAthallahAzhar/tokohobby-protos/pb/auth"
	productpb "github.com/RehanAthallahAzhar/tokohobby-protos/pb/product"
)

func main() {
	log := logger.NewLogger()
	log.Println("newlogger executed")

	cfg, err := configs.LoadConfig(log)
	if err != nil {
		log.Fatalf("FATAL: Gagal memuat konfigurasi: %v", err)
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
		log.Fatalf("DB connection error: %v", err)
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

	accountConn := createGrpcConnection(cfg.GRPC.AccountServiceAddress, log)
	defer accountConn.Close()

	accountClient := accountpb.NewAccountServiceClient(accountConn)
	authClient := authpb.NewAuthServiceClient(accountConn)
	authClientWrapper := account.NewAuthClientFromService(authClient, accountConn)

	// Publisher Rabbitmq
	rabbitMQURL := cfg.RabbitMQ.URL
	rabbitConn, err := amqp.Dial(rabbitMQURL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer rabbitConn.Close()

	rabbitChannel, err := rabbitConn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	defer rabbitChannel.Close()

	productsRepo := repositories.NewProductRepository(conn, sqlcQueries, log)
	cartsRepo := repositories.NewCartRepository(redisClient, log)
	validate := validator.New()
	productService := services.NewProductService(productsRepo, redisClient, validate, log)
	cartService := services.NewCartService(cartsRepo, productService, redisClient, accountClient, log)
	handler := handlers.NewHandler(productService, cartService, log)
	authMiddleware := customMiddleware.AuthMiddleware(authClientWrapper, log)

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
		AllowOrigins: []string{
			"http://localhost",
			"http://localhost:5173",
			"http://72.61.142.248",
		},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
	}))

	routes.InitRoutes(e, handler, authMiddleware)

	e.Logger.Fatal(e.Start(":" + cfg.Server.Port))
}

func createGrpcConnection(url string, log *logrus.Logger) *grpc.ClientConn {
	conn, err := grpc.NewClient(url, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to create gRPC client connection to %s: %v", url, err)
	}

	return conn
}
