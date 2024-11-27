package registry

import "time"

// ServerInfo represents the structure of server details
type ServerInfo struct {
	Type     string    `json:"type"`
	IP       string    `json:"ip"`
	Port     int       `json:"port"`
	ID       string    `json:"id"`
	LastSeen time.Time `json:"last_seen"`
	Group    string    `json:"group"`
}
