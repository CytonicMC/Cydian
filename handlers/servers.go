package handlers

import (
	"encoding/json"
	"log"

	"github.com/CytonicMC/Cydian/env"
	"github.com/CytonicMC/Cydian/servers"
	"github.com/nats-io/nats.go"
)

func RegisterServers(nc *nats.Conn, registry *servers.Registry) {
	registrationHandler(nc, registry)
	shutdownHandler(nc, registry)
	listHandler(nc, registry)
	proxyStartupHandler(nc, registry)
}

// registrationHandler sets up the NATS subscription for server registration
func registrationHandler(nc *nats.Conn, reg *servers.Registry) {
	const subject = "servers.register"

	_, err := nc.Subscribe(env.EnsurePrefixed(subject), func(msg *nats.Msg) {
		var serverInfo servers.ServerInfo
		if err := json.Unmarshal(msg.Data, &serverInfo); err != nil {
			log.Printf("Invalid message format: %s", msg.Data)
			return
		}

		// Add or update the server in the registry
		reg.AddOrUpdate(serverInfo)
		log.Printf("Registered server: %s", serverInfo.ID)

		// add them to the proxies at runtime
		NotifyProxiesOfStartup(nc, serverInfo)
	})
	if err != nil {
		log.Fatalf("Error subscribing to subject %s: %v", subject, err)
	}
	log.Printf("Listening for server registrations on subject '%s'", subject)
}

// ShutdownHandler sets up the NATS subscription for server shut-downs (Graceful ones, anyway.)
func shutdownHandler(nc *nats.Conn, reg *servers.Registry) {
	const subject = "servers.shutdown"

	_, err := nc.Subscribe(env.EnsurePrefixed(subject), func(msg *nats.Msg) {
		var serverInfo servers.ServerInfo
		if err := json.Unmarshal(msg.Data, &serverInfo); err != nil {
			log.Printf("Invalid message format: %s", msg.Data)
			return
		}

		// Add or update the server in the registry
		reg.Remove(serverInfo.ID)
		log.Printf("Removed server: %s", serverInfo.ID)

		// notify proxies that servers have been removed
		NotifyProxiesOfShutdown(nc, serverInfo)
	})
	if err != nil {
		log.Fatalf("Error subscribing to subject %s: %v", subject, err)
	}
	log.Printf("Listening for server shutdown on subject '%s'", subject)
}

func NotifyProxiesOfShutdown(nc *nats.Conn, serverInfo servers.ServerInfo) {
	const subject = "servers.proxy.shutdown.notify"

	data, err := json.Marshal(serverInfo)
	if err != nil {
		log.Printf("Failed to jsonify serverInfo")
		return
	}

	err = nc.Publish(env.EnsurePrefixed(subject), data)
	log.Printf("Notified proxies of shutdown for server: %s", serverInfo.ID)
	if err != nil {
		log.Printf("Failed to publish a server shutdown message: %v", err)
	}
}

func NotifyProxiesOfStartup(nc *nats.Conn, serverInfo servers.ServerInfo) {
	const subject = "servers.proxy.startup.notify"

	data, err := json.Marshal(serverInfo)
	if err != nil {
		log.Printf("Failed to jsonify serverInfo")
		return
	}

	err = nc.Publish(env.EnsurePrefixed(subject), data)
	log.Printf("published server startup proxy notification")
	if err != nil {
		log.Printf("Failed to publish a server startup message: %v", err)
	}
}

// ListHandler Handles NATS requests by replying will all the registered servers
func listHandler(nc *nats.Conn, reg *servers.Registry) {
	const subject = "servers.list"
	_, err := nc.Subscribe(env.EnsurePrefixed(subject), func(msg *nats.Msg) {
		// Fetch all registered all
		all := servers.ServerList{
			Servers: reg.GetAll(),
		}
		// Serialize to JSON
		response, err := json.Marshal(all)
		if err != nil {
			log.Printf("Error marshalling all list: %v", err)
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

func proxyStartupHandler(nc *nats.Conn, reg *servers.Registry) {
	const subject = "servers.proxy.startup"

	_, err := nc.Subscribe(env.EnsurePrefixed(subject), func(msg *nats.Msg) {
		// don't care about the message

		ack, err := json.Marshal(reg.GetAll())

		if err != nil {
			log.Printf("Error marshalling all list: %v", err)
			ack = []byte("ERROR: Failed to marshal all servers")
		}

		// either respond with the json data, or the error message
		if err1 := msg.Respond(ack); err1 != nil {
			log.Printf("Error sending acknowledgment: %v", err1)
		}

	})
	if err != nil {
		log.Fatalf("Error subscribing to subject %s: %v", subject, err)
	}
	log.Printf("Listening for proxy startup on subject '%s'", subject)
}
