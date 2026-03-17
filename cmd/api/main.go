package main

import (
	"log"

	"dscheckerapp/internal/config"
	"dscheckerapp/internal/db"
	"dscheckerapp/internal/router"
)

func main() {
	cfg := config.Load()

	database, err := db.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect db: %v", err)
	}
	defer database.Close()

	r := router.SetupRouter(database)

	log.Println("server started on :" + cfg.Port)

	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}