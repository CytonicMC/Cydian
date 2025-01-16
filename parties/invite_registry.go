package parties

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"log"
	"sync"
	"time"
)

// InviteRegistry to store active servers
type InviteRegistry struct {
	mu sync.Mutex
	// keyed by REQUEST uuid
	invites map[uuid.UUID]PartyInvite
	// First is sender, second is recipient
}

// NewInviteRegistry creates a new Registry instance
func NewInviteRegistry() *InviteRegistry {
	return &InviteRegistry{invites: make(map[uuid.UUID]PartyInvite)}
}

func (r *InviteRegistry) CreateInvite(sender uuid.UUID, party uuid.UUID, recipeint uuid.UUID) (*PartyInvite, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.containsPair(party, recipeint) {
		fmt.Printf("Party Invite registry already contains an invite to %s, for the party: %s", recipeint, party)
		return nil, errors.New(fmt.Sprintf("%s has already been invited to the party %s!", recipeint, party))
	}

	inviteUUID := uuid.New()
	invite := PartyInvite{
		ID:        inviteUUID,
		PartyID:   party,
		Recipient: recipeint,
		Sender:    sender,
		Expiry:    time.Now().Add(time.Second * 60),
	}

	// they expire after 60 seconds
	time.AfterFunc(time.Second*60, func() {
		r.expireInvite(inviteUUID)
	})

	r.invites[inviteUUID] = invite

	log.Printf("Added: %+v", invite)
	return &invite, nil
	//todo send messages
}

// Accept Accepts the request with the specified ID.
func (r *InviteRegistry) Accept(id uuid.UUID) (bool, PartyInvite) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.containsKey(id) {
		req := r.invites[id]
		delete(r.invites, id)
		log.Printf("Accepted invite: %+v", id)
		return true, req
	}
	log.Printf("Attempted to accept invalid invite: %+v", id)
	return false, PartyInvite{}
}

// GetAll returns all active servers
func (r *InviteRegistry) GetAll() []PartyInvite {
	r.mu.Lock()
	defer r.mu.Unlock()
	servers := make([]PartyInvite, 0, len(r.invites))
	for _, info := range r.invites {
		servers = append(servers, info)
	}
	return servers
}

func (r *InviteRegistry) Get(id uuid.UUID) PartyInvite {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.invites[id]
}

func (r *InviteRegistry) expireInvite(requestUUID uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.invites, requestUUID)
	//todo: broadcast expiry message
}

func (r *InviteRegistry) contains(invite PartyInvite) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, friendRequest := range r.invites {
		if invite == friendRequest {
			return true
		}
	}
	return false
}

func (r *InviteRegistry) containsPair(party uuid.UUID, recipient uuid.UUID) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, invite := range r.invites {
		if invite.Recipient == recipient && invite.PartyID == party {
			return true
		}
	}
	return false
}

func (r *InviteRegistry) containsKey(id uuid.UUID) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	for u := range r.invites {
		if id == u {
			return true
		}
	}
	return false
}
