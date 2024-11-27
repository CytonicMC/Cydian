package main

import (
	"github.com/CytonicMC/Cydian/handlers"
	"github.com/CytonicMC/Cydian/metrics"
	"github.com/CytonicMC/Cydian/servers/registry"
	"github.com/nats-io/nats.go"
	"log"
	"time"
)

func main() {
	// Initialize Prometheus metrics
	metrics.InitMetrics()
	metrics.ServeMetrics()

	// Connect to NATS server
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatalf("Error connecting to NATS: %v", err)
	}
	defer nc.Close()
	log.Println("Connected to NATS!")

	// Initialize the server registry
	reg := registry.NewRegistry()

	// Set up handler for server registration
	handlers.RegisterAll(nc, reg)

	// Periodic cleanup of stale servers
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			reg.HealthCheck(nc, 5*time.Second) // 5-second timeout per server
		}
	}()

	// Keep the service running
	select {}
}
