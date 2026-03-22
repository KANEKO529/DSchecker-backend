package main

import (
	"log"

	"dscheckerapp/internal/config"
	"dscheckerapp/internal/db"	
	"dscheckerapp/internal/lib"
	"dscheckerapp/internal/router"
)

func main() {
	cfg := config.Load()

	stripeCfg, err := lib.LoadStripeConfig()
	if err != nil {
		log.Fatalf("failed to load stripe config: %v", err)
	}

	lib.InitStripe(stripeCfg)

	database, err := db.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect db: %v", err)
	}
	defer database.Close()

	r := router.SetupRouter(database, stripeCfg)

	log.Println("server started on :" + cfg.Port)

	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}