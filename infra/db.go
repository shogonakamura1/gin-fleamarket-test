package infra

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func SetupDB() *gorm.DB {
	dbName := os.Getenv("DB_NAME")
	env := os.Getenv("ENV")

	// DB_NAMEが設定されている場合はPostgreSQLを使用
	// DB_NAME="postgres"の場合もPostgreSQL接続を許可（main.goで特別な処理があるため）
	if dbName != "" {
		// 本番環境ではsslmode=require、それ以外はsslmode=disable
		sslmode := "disable"
		if env == "prod" {
			sslmode = "require"
		}

		dsn := fmt.Sprintf(
			"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=Asia/Tokyo",
			os.Getenv("DB_HOST"),
			os.Getenv("DB_USER"),
			os.Getenv("DB_PASSWORD"),
			dbName,
			os.Getenv("DB_PORT"),
			sslmode,
		)

		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			panic("Failed to connect to database")
		}
		log.Println("Setup postgres database")
		return db
	}

	// デフォルトはSQLiteのインメモリ（テスト用）
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to database")
	}
	log.Println("Setup sqlite database (in-memory)")
	return db
}

// SetupTokenDB トークンブラックリスト用のSQLiteデータベース接続を設定
func SetupTokenDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("token_blacklist.db"), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to token blacklist database")
	}
	log.Println("Setup token blacklist SQLite database")
	return db
}
