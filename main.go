package main

import (
	"log"
	"time"

	"github.com/CytonicMC/Cydian/app"
	"github.com/CytonicMC/Cydian/env"
	"github.com/CytonicMC/Cydian/friends"
	"github.com/CytonicMC/Cydian/handlers"
	"github.com/CytonicMC/Cydian/metrics"
	"github.com/CytonicMC/Cydian/parties"
	"github.com/CytonicMC/Cydian/servers"
	"github.com/CytonicMC/Cydian/utils"
	"github.com/nats-io/nats.go"
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
	log.Printf("Connected to NATS! (Using url %s)\n", utils.NatsUrl())

	// Initialize the registries
	serverReg := servers.NewRegistry()
	friendReg := friends.NewRegistry(nc)
	partyReg := parties.NewPartyRegistry(nc)
	partyInviteReg := parties.NewInviteRegistry(nc, partyReg)

	instance := &app.Cydian{
		ServerRegistry:        serverReg,
		FriendRequestRegistry: friendReg,
		PartyInviteRegistry:   partyInviteReg,
		PartyRegistry:         partyReg,
	}

	// Set up handlers
	handlers.RegisterServers(nc, serverReg)
	handlers.RegisterFriends(nc, friendReg)
	handlers.RegisterPartyInvites(nc, partyInviteReg, instance)
	handlers.RegisterParties(nc, partyReg)
	handlers.RegisterInstances(nc)
	handlers.RegisterPlayerHandlers(nc, instance)

	// Periodic cleanup of stale servers
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			serverReg.HealthCheck(nc, 5*time.Second) // 5-second timeout per server
		}
	}()

	// Keep the service running
	log.Printf("Started Cydian in environment %s\n", env.Environment())
	select {}
}
