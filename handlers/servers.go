package handlers

import (
	"encoding/json"
	"github.com/CytonicMC/Cydian/servers/registry"
	"github.com/nats-io/nats.go"
	"log"
)

func RegisterAll(nc *nats.Conn, registry *registry.Registry) {
	RegisterHandler(nc, registry)
	ListHandler(nc, registry)
}

// RegisterHandler sets up the NATS subscription for server registration
func RegisterHandler(nc *nats.Conn, reg *registry.Registry) {
	const subject = "servers.register"

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

		// add them to the proxies at runtime
		NotifyProxiesOfStartup(nc, serverInfo)
	})
	if err != nil {
		log.Fatalf("Error subscribing to subject %s: %v", subject, err)
	}
	log.Printf("Listening for server registrations on subject '%s'", subject)
}

// ShutdownHandler sets up the NATS subscription for server shut-downs (Graceful ones, anyway.)
func ShutdownHandler(nc *nats.Conn, reg *registry.Registry) {
	const subject = "servers.shutdown"

	_, err := nc.Subscribe(subject, func(msg *nats.Msg) {
		var serverInfo registry.ServerInfo
		if err := json.Unmarshal(msg.Data, &serverInfo); err != nil {
			log.Printf("Invalid message format: %s", msg.Data)
			return
		}

		// Add or update the server in the registry
		reg.Remove(serverInfo.ID)

		// Respond with acknowledgment
		ack := []byte("Server removed successfully")
		if err := msg.Respond(ack); err != nil {
			log.Printf("Error sending acknowledgment: %v", err)
		}

		// notify proxies that servers have been removed
		NotifyProxiesOfShutdown(nc, serverInfo)
	})
	if err != nil {
		log.Fatalf("Error subscribing to subject %s: %v", subject, err)
	}
	log.Printf("Listening for server shutdown on subject '%s'", subject)
}

func NotifyProxiesOfShutdown(nc *nats.Conn, serverInfo registry.ServerInfo) {
	const subject = "servers.notify-proxies.shutdown"

	data, err := json.Marshal(serverInfo)
	if err != nil {
		log.Printf("Failed to jsonify serverInfo")
		return
	}

	err = nc.Publish(subject, data)
	if err != nil {
		log.Printf("Failed to publish a server shutdown message: %v", err)
	}
}

func NotifyProxiesOfStartup(nc *nats.Conn, serverInfo registry.ServerInfo) {
	const subject = "servers.notify-proxies.startup"

	data, err := json.Marshal(serverInfo)
	if err != nil {
		log.Printf("Failed to jsonify serverInfo")
		return
	}

	err = nc.Publish(subject, data)
	if err != nil {
		log.Printf("Failed to publish a server startup message: %v", err)
	}
}

// ListHandler Handles NATS requests by replying will all the registered servers
func ListHandler(nc *nats.Conn, reg *registry.Registry) {
	const subject = "servers.list"
	_, err := nc.Subscribe(subject, func(msg *nats.Msg) {
		// Fetch all registered servers
		servers := reg.GetAll()

		// Serialize to JSON
		response, err := json.Marshal(servers)
		if err != nil {
			log.Printf("Error marshalling servers list: %v", err)
			err := msg.Respond([]byte("Error generating server list"))
			if err != nil {
				log.Printf("Error sending acknowledgment ;-; -> %v", err)
				return
			}
			return
		}

		// Send response
		if err := msg.Respond(response); err != nil {
			log.Printf("Error responding to server list request: %v", err)
		}
	})
	if err != nil {
		log.Fatalf("Error subscribing to subject %s: %v", subject, err)
	}
	log.Printf("Listening for server list requests on subject '%s'", subject)
}
