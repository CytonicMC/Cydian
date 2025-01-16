package servers

import (
	"encoding/json"
	"time"
)

// ServerInfo represents the structure of server details
type ServerInfo struct {
	Type     string     `json:"type"`
	IP       string     `json:"ip"`
	Port     int        `json:"port"`
	ID       string     `json:"id"`
	LastSeen *time.Time `json:"last_seen"` // pointer indicates the value may be null.
	Group    string     `json:"group"`
}

func (s ServerInfo) MarshalJSON() ([]byte, error) {
	type Alias ServerInfo
	return json.Marshal(&struct {
		LastSeen interface{} `json:"last_seen"`
		*Alias
	}{
		LastSeen: func() interface{} {
			if s.LastSeen == nil {
				return nil
			}
			return s.LastSeen.UTC().Format(time.RFC3339)
		}(),
		Alias: (*Alias)(&s),
	})
}
