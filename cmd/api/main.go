package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/redis/go-redis/v9"
	_ "github.com/lib/pq" // Postgres driver

	delivery "github.com/FilipeAphrody/sentinel-auth/internal/delivery/http"
	"github.com/FilipeAphrody/sentinel-auth/internal/repository"
	"github.com/FilipeAphrody/sentinel-auth/internal/usecase"
)

func main() {
	// 1. Initialize Echo instance
	e := echo.New()

	// 2. Load Configuration from Environment Variables
	// In a real Staff-level app, use a config struct/library (like Viper or cleanenv)
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "staff-level-super-secret-key" // Fallback for local dev
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		dbURL = "postgres://user:password@localhost:5432/sentinel?sslmode=disable"
	}

	redisAddr := os.Getenv("REDIS_URL")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	// 3. Initialize Infrastructure (Database & Cache)
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Critical: failed to connect to database: %v", err)
	}
	defer db.Close()

	// Verify DB connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Critical: database is unreachable: %v", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	defer rdb.Close()

	// 4. Initialize Clean Architecture Layers (Dependency Injection)
	userRepo := repository.NewPostgresUserRepo(db)
	tokenRepo := repository.NewRedisTokenRepo(rdb)
	authUsecase := usecase.NewAuthUsecase(userRepo, tokenRepo, jwtSecret)

	// 5. Global Middlewares
	e.Use(middleware.Logger())    // Request logging
	e.Use(middleware.Recover())   // Panic recovery
	e.Use(middleware.CORS())      // Cross-Origin Resource Sharing
	e.Use(middleware.Secure())    // Protection against XSS, Content-Type Sniffing, etc.
	e.Use(middleware.BodyLimit("1M")) // Prevent large payload attacks

	// 6. Route Definition
	v1 := e.Group("/v1")

	// Public Routes (Registration/Login)
	delivery.NewAuthHandler(v1, authUsecase)

	// Protected Routes (Require valid JWT)
	protected := v1.Group("")
	protected.Use(delivery.JWTMiddleware(jwtSecret))
	
	// MFA Setup & Management (Now secured by the middleware)
	delivery.NewMFAHandler(protected, authUsecase)

	// Health Check for monitoring/LBs
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, echo.Map{
			"status":  "healthy",
			"time":    time.Now().Format(time.RFC3339),
			"version": "1.0.0",
		})
	})

	// 7. Start Server with Graceful Shutdown
	// This ensures in-flight requests finish before the process exits
	go func() {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		
		fmt.Printf("🛡️ Sentinel Auth Server starting on port %s...\n", port)
		if err := e.Start(":" + port); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for OS interrupt signal (SIGINT or SIGTERM)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	fmt.Println("\n⚠️ Shutting down server gracefully...")
	
	// Set a timeout for the shutdown process
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		log.Fatalf("Graceful shutdown failed: %v", err)
	}
	
	fmt.Println("🛑 Server stopped.")
}