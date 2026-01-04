package parties

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/CytonicMC/Cydian/env"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

// InviteRegistry to store active servers
type InviteRegistry struct {
	mu sync.Mutex
	// keyed by REQUEST uuid
	invites         map[UUID]PartyInvite
	expiryFunctions map[UUID]*time.Timer
	partyRegistry   *PartyRegistry
	nc              *nats.Conn
}

// NewInviteRegistry creates a new Registry instance
func NewInviteRegistry(conn *nats.Conn, registry *PartyRegistry) *InviteRegistry {
	return &InviteRegistry{
		invites:         make(map[UUID]PartyInvite),
		expiryFunctions: make(map[UUID]*time.Timer),
		nc:              conn,
		partyRegistry:   registry,
	}
}

func (r *InviteRegistry) CreateInvite(sender UUID, party UUID, recipeint UUID) (*PartyInvite, string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	partyObj := r.partyRegistry.GetParty(party)
	if partyObj != nil { // means it's a new party
		if partyObj.IsInParty(recipeint) {
			return nil, "ERR_ALREADY_IN_PARTY"
		}
		if !partyObj.OpenInvites && partyObj.CurrentLeader != sender {
			return nil, "ERR_NO_PERMISSION"
		}
	}
	if r.containsPairInternal(party, recipeint) {
		fmt.Printf("Party Invite registry already contains an invite to %s, for the party: %s", recipeint, party)
		return nil, "ERR_ALREADY_INVITED"
	}

	inviteUUID := UUID(uuid.New())
	invite := PartyInvite{
		ID:        inviteUUID,
		PartyID:   party,
		Recipient: recipeint,
		SenderID:  sender,
		Expiry:    time.Now().Add(time.Second * 60),
	}

	// they expire after 60 seconds
	r.expiryFunctions[inviteUUID] = time.AfterFunc(time.Second*60, func() {
		r.expireInvite(inviteUUID)
	})

	r.invites[inviteUUID] = invite
	r.partyRegistry.TrackInvite(party, invite)

	log.Printf("Party (%+v) invite sent: %+v", party, invite)
	return &invite, ""
}

// Accept Accepts the request with the specified ID.
func (r *InviteRegistry) Accept(id UUID) (bool, *PartyInvite) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.containsKeyInternal(id) {
		log.Printf("Attempted to accept invalid invite: %+v", id)
		return false, nil
	}

	req := r.invites[id]
	delete(r.invites, id)
	r.expiryFunctions[id].Stop()
	delete(r.expiryFunctions, id)

	if !r.partyRegistry.containsKey(req.PartyID) {
		log.Printf("Attempted to accept an invite(%v) to a non-existant party: %+v", id, req.PartyID)
		return false, nil
	}

	r.partyRegistry.JoinParty(req.PartyID, req.Recipient, true)

	log.Printf("Party invite(%+v) accepted", id)
	return true, &req

}

// GetAll returns all active invites
func (r *InviteRegistry) GetAll() []PartyInvite {
	r.mu.Lock()
	defer r.mu.Unlock()
	servers := make([]PartyInvite, 0, len(r.invites))
	for _, info := range r.invites {
		servers = append(servers, info)
	}
	return servers
}

func (r *InviteRegistry) Get(id UUID) PartyInvite {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.invites[id]
}

func (r *InviteRegistry) expireInvite(requestUUID UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.containsKeyInternal(requestUUID) {
		return
	}

	invite := r.invites[requestUUID]
	delete(r.invites, requestUUID)

	if !r.partyRegistry.RemoveInvite(invite.PartyID, invite.ID) {
		return // prevents sending notices if it was somehow already accepted
	}

	obj := &PartyInviteExpirePacket{
		RequestID: requestUUID, PartyID: invite.PartyID,
		Recipient: invite.Recipient, SenderID: invite.SenderID,
	}
	serailized, err1 := json.Marshal(obj)
	if err1 != nil {
		log.Printf("Error marshalling invite (%v) expiry: %v", requestUUID, err1)
		return
	}
	err := r.nc.Publish(env.EnsurePrefixed("parties.invite.expire"), serailized)
	if err != nil {
		log.Printf("Error publishing invite (%v) expiry: %v", requestUUID, err)
	}
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

func (r *InviteRegistry) containsPair(party UUID, recipient UUID) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.containsPairInternal(party, recipient)
}

func (r *InviteRegistry) containsPairInternal(party UUID, recipient UUID) bool {
	for _, invite := range r.invites {
		if invite.Recipient == recipient && invite.PartyID == party {
			return true
		}
	}
	return false
}

func (r *InviteRegistry) containsKey(id UUID) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.containsKeyInternal(id)
}

func (r *InviteRegistry) containsKeyInternal(id UUID) bool {
	for u := range r.invites {
		if id == u {
			return true
		}
	}
	return false
}
