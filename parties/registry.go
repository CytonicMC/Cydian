package parties

import "sync"

type PartyRegistry struct {
	mu      sync.Mutex
	parties map[string]Party
}

func NewPartyRegistry() *PartyRegistry {
	return &PartyRegistry{
		mu:      sync.Mutex{},
		parties: make(map[string]Party),
	}
}
