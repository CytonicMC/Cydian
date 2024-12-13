package servers

import "time"

// ServerInfo represents the structure of server details
type ServerInfo struct {
	Type     string     `json:"type"`
	IP       string     `json:"ip"`
	Port     int        `json:"port"`
	ID       string     `json:"id"`
	LastSeen *time.Time `json:"last_seen"` // pointer indicates the value may be null.
}
