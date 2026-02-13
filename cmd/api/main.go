package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/redis/go-redis/v9"

	delivery "github.com/FilipeAphrody/sentinel-auth/internal/delivery/http"
	"github.com/FilipeAphrody/sentinel-auth/internal/repository"
	"github.com/FilipeAphrody/sentinel-auth/internal/usecase"
	
	_ "github.com/lib/pq" // Postgres driver
)

func main() {
	// 1. Setup Framework
	e := echo.New()

	// 2. Load Configuration from Environment
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		dbURL = "postgres://user:password@localhost:5432/sentinel?sslmode=disable"
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "localhost:6379"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "staff-level-super-secret-key"
	}

	// 3. Initialize Infrastructure (Persistence)
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr: redisURL,
	})
	defer rdb.Close()

	// 4. Initialize Repositories
	userRepo := repository.NewPostgresUserRepo(db)
	tokenRepo := repository.NewRedisTokenRepo(rdb)

	// 5. Initialize Business Logic (Usecases)
	authUsecase := usecase.NewAuthUsecase(userRepo, tokenRepo, jwtSecret)

	// 6. Global Middlewares
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())
	e.Use(middleware.Secure())

	// 7. Register Delivery Handlers (Routes)
	v1 := e.Group("/v1")
	delivery.NewAuthHandler(v1, authUsecase)
	delivery.NewMFAHandler(v1, authUsecase)

	// 8. Health Check
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, echo.Map{
			"status":  "healthy",
			"version": "1.0.0",
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	// 9. Start Server with Graceful Shutdown
	go func() {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		
		log.Printf("Starting Sentinel Auth Server on port %s", port)
		if err := e.Start(":" + port); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Shutting down the server due to error: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	
	log.Println("Server exiting")
}