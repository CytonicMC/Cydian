package handlers

import (
	"Cydian/servers/registry"
	"encoding/json"
	"github.com/nats-io/nats.go"
	"log"
)

// RegisterHandler sets up the NATS subscription for server registration
func RegisterHandler(nc *nats.Conn, reg *registry.Registry) {
	const subject = "service.register"

	_, err := nc.Subscribe(subject, func(msg *nats.Msg) {
		var serverInfo registry.ServerInfo
		if err := json.Unmarshal(msg.Data, &serverInfo); err != nil {
			log.Printf("Invalid message format: %s", msg.Data)
			return
		}

		// Add or update the server in the registry
		reg.AddOrUpdate(serverInfo)

		// Respond with acknowledgment
		ack := []byte("Server registered successfully")
		if err := msg.Respond(ack); err != nil {
			log.Printf("Error sending acknowledgment: %v", err)
		}
	})
	if err != nil {
		log.Fatalf("Error subscribing to subject %s: %v", subject, err)
	}
	log.Printf("Listening for server registrations on subject '%s'", subject)
}
