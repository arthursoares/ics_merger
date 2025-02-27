package main

import (
	"log"
	"time"

	"github.com/arthur/ical_merger/internal/app"
	"github.com/arthur/ical_merger/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	merger := app.NewMerger(cfg)
	
	// Do initial merge
	if err := merger.Merge(); err != nil {
		log.Printf("Initial merge failed: %v", err)
	}

	// Set up periodic merges
	ticker := time.NewTicker(time.Duration(cfg.SyncIntervalMinutes) * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Println("Starting periodic merge")
			if err := merger.Merge(); err != nil {
				log.Printf("Periodic merge failed: %v", err)
			}
		}
	}
}