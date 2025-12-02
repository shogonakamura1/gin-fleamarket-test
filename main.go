package main

import (
	"context"
	"fmt"
	"gin-fleamarket/controllers"
	"gin-fleamarket/infra"
	"gin-fleamarket/middlewares"
	"gin-fleamarket/models"
	"gin-fleamarket/repositories"
	"gin-fleamarket/services"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupRouter(db *gorm.DB) *gin.Engine {

	itemRepository := repositories.NewItemRepository(db)
	itemService := services.NewItemService(itemRepository)
	itemController := controllers.NewItemController(itemService)

	authRepository := repositories.NewAuthRepository(db)
	tokenDB := infra.SetupTokenDB()
	tokenRepository := repositories.NewTokenRepository(tokenDB)
	authService := services.NewAuthService(authRepository, tokenRepository)
	authController := controllers.NewAuthController(authService)

	// トークンブラックリスト用のマイグレーション
	if os.Getenv("AUTO_MIGRATE") == "true" {
		if err := tokenDB.AutoMigrate(&models.BlacklistedToken{}); err != nil {
			log.Printf("Failed to migrate token blacklist database: %v", err)
		}
	}

	r := gin.Default()
	r.Use(cors.Default())
	itemRouter := r.Group("/items")
	itemRouterWithAuth := r.Group("/items", middlewares.AuthMiddleware(authService))
	itemRouterWithAdminAuth := r.Group("/items", middlewares.AuthMiddleware(authService), middlewares.RoleBasedAccessControl("admin"))
	authRouter := r.Group("/auth")

	itemRouter.GET("", itemController.FindAll)
	itemRouterWithAuth.GET("/:id", itemController.FindById)
	itemRouterWithAuth.POST("", itemController.Create)
	itemRouterWithAuth.PUT("/:id", itemController.Update)
	itemRouterWithAdminAuth.DELETE("/:id", itemController.Delete)

	authRouter.POST("/signup", authController.Signup)
	authRouter.POST("/login", authController.Login)
	authRouter.POST("/logout", authController.Logout)

	return r
}

var (
	globalDB   *gorm.DB
	dbReady    = make(chan struct{})
	dbInitOnce sync.Once
)

func initDB() *gorm.DB {
	infra.Initialize()

	db := infra.SetupDB()

	targetDBName := "fleamarket"
	currentDBName := os.Getenv("DB_NAME")

	if currentDBName == "postgres" {
		var exists int
		db.Raw("SELECT 1 FROM pg_database WHERE datname = ?", targetDBName).Scan(&exists)
		if exists == 0 {
			if err := db.Exec(fmt.Sprintf("CREATE DATABASE %s", targetDBName)).Error; err != nil {
				log.Printf("Failed to create database: %v", err)
			} else {
				log.Printf("Created database: %s", targetDBName)
			}
		}

		dbHost := os.Getenv("DB_HOST")
		dbUser := os.Getenv("DB_USER")
		dbPassword := os.Getenv("DB_PASSWORD")
		dbPort := os.Getenv("DB_PORT")

		log.Printf("Connecting to fleamarket database: host=%s, user=%s, dbname=%s, port=%s",
			dbHost, dbUser, targetDBName, dbPort)

		// 本番環境ではsslmode=require、それ以外はsslmode=disable
		env := os.Getenv("ENV")
		sslmode := "disable"
		if env == "prod" {
			sslmode = "require"
		}

		dsn := fmt.Sprintf(
			"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=Asia/Tokyo connect_timeout=10",
			dbHost,
			dbUser,
			dbPassword,
			targetDBName,
			dbPort,
			sslmode,
		)

		var err error
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			log.Printf("Failed to connect to fleamarket database: %v", err)
			log.Printf("Connection string (without password): host=%s, user=%s, dbname=%s, port=%s",
				dbHost, dbUser, targetDBName, dbPort)
			panic(fmt.Sprintf("Failed to connect to fleamarket database: %v", err))
		}
		log.Printf("Successfully connected to database: %s", targetDBName)
	} else {
		log.Printf("Using existing database: %s", currentDBName)
	}

	if os.Getenv("AUTO_MIGRATE") == "true" {
		if err := db.AutoMigrate(&models.User{}, &models.Item{}); err != nil {
			panic("Failed to migrate database")
		}

		// トークンブラックリスト用のSQLiteデータベースのマイグレーション
		tokenDB := infra.SetupTokenDB()
		if err := tokenDB.AutoMigrate(&models.BlacklistedToken{}); err != nil {
			log.Printf("Failed to migrate token blacklist database: %v", err)
		}
	}

	return db
}

func main() {
	isLambda := os.Getenv("AWS_LAMBDA_RUNTIME_API") != ""

	if isLambda {
		log.Println("Lambda environment detected, initializing database asynchronously...")

		port := os.Getenv("PORT")
		if port == "" {
			port = os.Getenv("AWS_LWA_PORT")
		}
		if port == "" {
			port = "8080"
		}

		r := gin.New()
		r.Use(gin.Logger())
		r.Use(gin.Recovery())
		r.Use(cors.Default())

		var routerMutex sync.RWMutex
		var actualRouter *gin.Engine

		r.GET("/", func(c *gin.Context) {
			log.Println("Health check endpoint called (root path)")
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})
		r.GET("/health", func(c *gin.Context) {
			log.Println("Health check endpoint called")
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		handler := func(c *gin.Context) {
			log.Printf("Request received: %s %s", c.Request.Method, c.Request.URL.Path)
			select {
			case <-dbReady:
				routerMutex.RLock()
				if actualRouter == nil {
					routerMutex.RUnlock()
					routerMutex.Lock()
					if actualRouter == nil {
						actualRouter = setupRouter(globalDB)
						log.Println("Router initialized with database connection")
					}
					routerMutex.Unlock()
				} else {
					routerMutex.RUnlock()
				}
				routerMutex.RLock()
				actualRouter.ServeHTTP(c.Writer, c.Request)
				routerMutex.RUnlock()
			case <-time.After(10 * time.Second):
				log.Println("Database connection timeout")
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database connection timeout"})
			}
		}

		r.NoRoute(handler)

		srv := &http.Server{
			Addr:         ":" + port,
			Handler:      r,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		}

		serverReady := make(chan bool, 1)
		go func() {
			log.Printf("Starting server on port %s (Lambda environment)", port)
			ln, err := net.Listen("tcp", ":"+port)
			if err != nil {
				log.Fatalf("Failed to listen on port %s: %v", port, err)
			}
			serverReady <- true
			if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Failed to start server: %v", err)
			}
		}()

		<-serverReady
		time.Sleep(50 * time.Millisecond)
		log.Printf("Server started on port %s, ready to handle requests (database connecting in background)", port)

		go func() {
			dbInitOnce.Do(func() {
				globalDB = initDB()
				close(dbReady)
				log.Println("Database connection established")
			})
		}()

		select {}
	} else {
		db := initDB()
		r := setupRouter(db)

		port := os.Getenv("PORT")
		if port == "" {
			port = os.Getenv("AWS_LWA_PORT")
		}
		if port == "" {
			port = "8080"
		}

		srv := &http.Server{
			Addr:         ":" + port,
			Handler:      r,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		}

		go func() {
			log.Printf("Starting server on port %s (local environment)", port)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Failed to start server: %v", err)
			}
		}()

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		log.Println("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Fatal("Server forced to shutdown:", err)
		}
		log.Println("Server exited")
	}
}
