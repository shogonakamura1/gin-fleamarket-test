package main

import (
	"gin-fleamarket/infra"
	"gin-fleamarket/models"
)

func main() {
	infra.Initialize()
	db := infra.SetupDB()

	if err := db.AutoMigrate(&models.Item{}, &models.User{}); err != nil {
		panic("Failed to migrate database")
	}

	// トークンブラックリスト用のSQLiteデータベースのマイグレーション
	tokenDB := infra.SetupTokenDB()
	if err := tokenDB.AutoMigrate(&models.BlacklistedToken{}); err != nil {
		panic("Failed to migrate token blacklist database")
	}
}
