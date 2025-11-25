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
	authService := services.NewAuthService(authRepository)
	authController := controllers.NewAuthController(authService)

	r := gin.Default()
	r.Use(cors.Default())
	itemRouter := r.Group("/items")
	itemRouterWithAuth := r.Group("/items", middlewares.AuthMiddleware(authService))
	authRouter := r.Group("/auth")

	itemRouter.GET("", itemController.FindAll)
	itemRouterWithAuth.GET("/:id", itemController.FindById)
	itemRouterWithAuth.POST("", itemController.Create)
	itemRouterWithAuth.PUT("/:id", itemController.Update)
	itemRouterWithAuth.DELETE("/:id", itemController.Delete)

	authRouter.POST("/signup", authController.Signup)
	authRouter.POST("/login", authController.Login)

	return r
}

var (
	globalDB   *gorm.DB
	dbReady    = make(chan struct{})
	dbInitOnce sync.Once
)

func initDB() *gorm.DB {
	infra.Initialize()

	// まずpostgresに接続してデータベースを作成
	db := infra.SetupDB()

	// DB_NAMEがpostgresの場合のみ、fleamarketデータベースを作成
	targetDBName := "fleamarket"
	currentDBName := os.Getenv("DB_NAME")

	if currentDBName == "postgres" {
		// fleamarketデータベースが存在しない場合は作成
		var exists int
		db.Raw("SELECT 1 FROM pg_database WHERE datname = ?", targetDBName).Scan(&exists)
		if exists == 0 {
			if err := db.Exec(fmt.Sprintf("CREATE DATABASE %s", targetDBName)).Error; err != nil {
				log.Printf("Failed to create database: %v", err)
			} else {
				log.Printf("Created database: %s", targetDBName)
			}
		}

		// fleamarketデータベースに接続し直す
		dbHost := os.Getenv("DB_HOST")
		dbUser := os.Getenv("DB_USER")
		dbPassword := os.Getenv("DB_PASSWORD")
		dbPort := os.Getenv("DB_PORT")

		log.Printf("Connecting to fleamarket database: host=%s, user=%s, dbname=%s, port=%s",
			dbHost, dbUser, targetDBName, dbPort)

		dsn := fmt.Sprintf(
			"host=%s user=%s password=%s dbname=%s port=%s sslmode=require TimeZone=Asia/Tokyo connect_timeout=10",
			dbHost,
			dbUser,
			dbPassword,
			targetDBName,
			dbPort,
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

	// Run AutoMigrate only when explicitly enabled
	if os.Getenv("AUTO_MIGRATE") == "true" {
		if err := db.AutoMigrate(&models.User{}, &models.Item{}); err != nil {
			panic("Failed to migrate database")
		}
	}

	return db
}

func main() {
	// Lambda環境かどうかを検出
	isLambda := os.Getenv("AWS_LAMBDA_RUNTIME_API") != ""

	if isLambda {
		// Lambda環境: データベース接続を非同期で行い、サーバーを先に起動
		log.Println("Lambda environment detected, initializing database asynchronously...")

		// Lambda Web Adapter用のポート設定
		port := os.Getenv("PORT")
		if port == "" {
			port = os.Getenv("AWS_LWA_PORT")
		}
		if port == "" {
			port = "8080"
		}

		// ルーターをセットアップ（データベース接続が完了するまで待機するミドルウェアを含む）
		// gin.New()を使用して、LoggerとRecoveryミドルウェアを手動で追加（高速化のため）
		r := gin.New()
		r.Use(gin.Logger())
		r.Use(gin.Recovery())
		r.Use(cors.Default())

		// データベース接続が完了したらルーターをセットアップ
		// ミドルウェア内でデータベース接続を待機し、ルーターを動的に設定
		var routerMutex sync.RWMutex
		var actualRouter *gin.Engine

		// ヘルスチェックエンドポイント（データベース接続を待たない）
		// Lambda Web Adapterはルートパス（/）でヘルスチェックを行う
		r.GET("/", func(c *gin.Context) {
			log.Println("Health check endpoint called (root path)")
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})
		r.GET("/health", func(c *gin.Context) {
			log.Println("Health check endpoint called")
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		// すべてのリクエストを処理するハンドラー
		handler := func(c *gin.Context) {
			log.Printf("Request received: %s %s", c.Request.Method, c.Request.URL.Path)
			// データベース接続が完了するまで待機
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
				// 実際のルーターでリクエストを処理
				routerMutex.RLock()
				actualRouter.ServeHTTP(c.Writer, c.Request)
				routerMutex.RUnlock()
			case <-time.After(10 * time.Second):
				log.Println("Database connection timeout")
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database connection timeout"})
			}
		}

		// すべてのリクエストを処理するハンドラー（NoRouteのみを使用）
		r.NoRoute(handler)

		// http.Serverを使用して明示的にサーバーを起動
		srv := &http.Server{
			Addr:         ":" + port,
			Handler:      r,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		}

		// サーバーを先に起動（データベース接続を待たない）
		// Lambda Web Adapterのヘルスチェックに即座に応答できるように、サーバーを非同期で起動
		serverReady := make(chan bool, 1)
		go func() {
			log.Printf("Starting server on port %s (Lambda environment)", port)
			// サーバーがリッスン状態になったことを通知
			ln, err := net.Listen("tcp", ":"+port)
			if err != nil {
				log.Fatalf("Failed to listen on port %s: %v", port, err)
			}
			serverReady <- true
			if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Failed to start server: %v", err)
			}
		}()

		// サーバーがリッスン状態になるまで待つ（Lambda Web Adapterのヘルスチェックに応答できるように）
		<-serverReady
		// サーバーが完全にリッスン状態になるまで少し待つ
		time.Sleep(50 * time.Millisecond)
		log.Printf("Server started on port %s, ready to handle requests (database connecting in background)", port)

		// データベース接続を非同期で開始（サーバー起動の後）
		go func() {
			dbInitOnce.Do(func() {
				globalDB = initDB()
				close(dbReady)
				log.Println("Database connection established")
			})
		}()

		// Lambda環境では、サーバーを起動したままブロックしない
		select {} // 無限に待機（Lambda関数が終了するまで）
	} else {
		// ローカル環境: 通常の初期化
		db := initDB()
		r := setupRouter(db)

		// Lambda Web Adapter用のポート設定
		port := os.Getenv("PORT")
		if port == "" {
			port = os.Getenv("AWS_LWA_PORT")
		}
		if port == "" {
			port = "8080"
		}

		// http.Serverを使用して明示的にサーバーを起動
		srv := &http.Server{
			Addr:         ":" + port,
			Handler:      r,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		}

		// ローカル環境: 通常の起動とシグナルハンドリング
		go func() {
			log.Printf("Starting server on port %s (local environment)", port)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Failed to start server: %v", err)
			}
		}()

		// シグナルハンドリング
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
