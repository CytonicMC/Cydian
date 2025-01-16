package main

import (
	"github.com/CytonicMC/Cydian/friends"
	"github.com/CytonicMC/Cydian/handlers"
	"github.com/CytonicMC/Cydian/metrics"
	"github.com/CytonicMC/Cydian/parties"
	"github.com/CytonicMC/Cydian/servers"
	"github.com/CytonicMC/Cydian/utils"
	"github.com/nats-io/nats.go"
	"log"
	"time"
)

func main() {
	// Initialize Prometheus metrics
	metrics.InitMetrics()
	metrics.ServeMetrics()

	// Connect to NATS server
	nc, err := nats.Connect(utils.NatsUrl())
	if err != nil {
		log.Fatalf("Error connecting to NATS: %v", err)
	}
	defer nc.Close()
	log.Println("Connected to NATS!")

	// Initialize the registries
	serverReg := servers.NewRegistry()
	friendReg := friends.NewRegistry(nc)
	partyInviteReg := parties.NewInviteRegistry()
	partyReg := parties.NewPartyRegistry()

	// Set up handlers
	handlers.RegisterServers(nc, serverReg)
	handlers.RegisterFriends(nc, friendReg)
	handlers.RegisterPartyInvites(nc, partyInviteReg)
	handlers.RegisterParties(nc, partyReg)

	// Periodic cleanup of stale servers
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			serverReg.HealthCheck(nc, 5*time.Second) // 5-second timeout per server
		}
	}()

	// Keep the service running
	select {}
}
