package servers

import (
	"encoding/json"
	"github.com/CytonicMC/Cydian/utils"
	"github.com/nats-io/nats.go"
	"log"
	"sync"
	"time"
)

// Registry to store active servers
type Registry struct {
	mu      sync.Mutex
	servers map[string]ServerInfo
}

// NewRegistry creates a new Registry instance
func NewRegistry() *Registry {
	return &Registry{servers: make(map[string]ServerInfo)}
}

// AddOrUpdate adds or updates server information in the registry
func (r *Registry) AddOrUpdate(info ServerInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()
	info.LastSeen = utils.PointerNow()
	r.servers[info.ID] = info
	log.Printf("Registered/Updated server: %+v", info)
}

func (r *Registry) Remove(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.servers, id)
	log.Printf("Removed server: %+v", id)
}

// Cleanup removes stale servers from the registry
func (r *Registry) Cleanup(timeout time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, info := range r.servers {
		if time.Since(*info.LastSeen) > timeout {
			log.Printf("Removing stale server: %s", id)
			delete(r.servers, id)
		}
	}
}

// GetAll returns all active servers
func (r *Registry) GetAll() []ServerInfo {
	r.mu.Lock()
	defer r.mu.Unlock()
	servers := make([]ServerInfo, 0, len(r.servers))
	for _, info := range r.servers {
		servers = append(servers, info)
	}
	return servers
}

// HealthCheck performs health checks on all servers in the registry
// todo: IMPLEMENT HEALTH CHECKS INTO CYTOSIS AND CYNDER
func (r *Registry) HealthCheck(nc *nats.Conn, timeout time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for id, server := range r.servers {
		subject := "health.check." + id
		msg, err := nc.Request(subject, nil, timeout)
		if err != nil {
			log.Printf("Server %s is unresponsive, removing from registry: %v", id, err)

			serverInfo := r.servers[id]

			data, err := json.Marshal(serverInfo)
			if err != nil {
				log.Printf("Failed to jsonify serverInfo")
				return
			}

			err = nc.Publish("servers.proxy.shutdown.notify", data)
			log.Printf("Notified proxies of shutdown for server '%s' due to health check failure", serverInfo.ID)

			delete(r.servers, id)
			continue
		}

		log.Printf("Received health response from server %s: %s", id, string(msg.Data))
		server.LastSeen = utils.PointerNow() // Update last seen time on success
		r.servers[id] = server
	}
}
