package main

import (
	"log"

	_ "github.com/mattn/go-sqlite3" // registers the sqlite3 driver used by LockStore
)

func main() {
	cfg, err := loadConfiguration()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}
	configureLogging(cfg.Debug)

	store, err := NewLockStore(DatabaseFile)
	if err != nil {
		log.Fatalf("Database initialization error: %v", err)
	}
	defer func() {
		if closeErr := store.Close(); closeErr != nil {
			log.Printf("Error closing database: %v", closeErr)
		}
	}()

	bot := NewBot(cfg, store)
	if err := bot.Run(); err != nil {
		log.Fatalf("Bot error: %v", err)
	}
}
