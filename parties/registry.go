package parties

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/CytonicMC/Cydian/env"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

type PartyRegistry struct {
	mu          sync.Mutex
	parties     map[UUID]Party
	disconnects map[UUID]context.CancelFunc
	nc          *nats.Conn
}

func NewPartyRegistry(nc *nats.Conn) *PartyRegistry {
	return &PartyRegistry{
		mu:          sync.Mutex{},
		parties:     make(map[UUID]Party),
		disconnects: make(map[UUID]context.CancelFunc),
		nc:          nc,
	}
}

func (r *PartyRegistry) CreateParty(id UUID, owner UUID, initialInvite *PartyInvite) Party {
	party := Party{
		ID:            id,
		CurrentLeader: owner,
		Moderators:    []UUID{},
		Members:       []UUID{},
		Open:          false,
		OpenInvites:   false,
		Muted:         false,
		ActiveInvites: make(map[UUID]PartyInvite),
	}
	if initialInvite != nil {
		party.ActiveInvites[initialInvite.ID] = *initialInvite
	}
	r.mu.Lock()
	r.parties[party.ID] = party
	r.mu.Unlock()

	msg, err := json.Marshal(PartyCreatePacket{Party: party})
	if err != nil {
		log.Printf("Failed to marshal party create packet: %v", err)
	}
	_ = r.nc.Publish(env.EnsurePrefixed("party.create.notify"), msg)
	log.Printf("create notification")
	return party
}

func (r *PartyRegistry) TrackInvite(partyID UUID, invite PartyInvite) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.containsKeyInternal(partyID) {
		return
	}

	party := r.parties[partyID]
	invites := party.ActiveInvites
	invites[invite.ID] = invite
	party.ActiveInvites = invites
	r.parties[partyID] = party
}

func (r *PartyRegistry) RemoveInvite(partyID UUID, inviteID UUID) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.containsKeyInternal(partyID) {
		log.Printf("Failed to remove invite: registry does not contain party: %s", partyID)
		return false
	}

	party := r.parties[partyID]
	invites := party.ActiveInvites
	_, ok := invites[inviteID]
	if !ok {
		return false
	}
	delete(invites, inviteID)
	party.ActiveInvites = invites
	r.parties[partyID] = party

	// check for an empty party and remove it
	if len(party.Members) == 0 && len(party.Moderators) == 0 && len(party.ActiveInvites) == 0 {
		r.disbandForEmptyInternal(partyID)
		return false
	}
	return true
}

func (r *PartyRegistry) disbandForEmptyInternal(partyID UUID) {
	delete(r.parties, partyID)

	log.Printf("Party %s has been disbanded due to being empty", partyID)

	msg, _ := json.Marshal(&PartyOnePlayerPacket{
		PartyID:  partyID,
		PlayerID: UUID(uuid.Nil),
	})
	err := r.nc.Publish(env.EnsurePrefixed("party.disband.notify.empty"), msg)
	if err != nil {
		log.Printf("Failed to broadcast party disband: %v", err)
	}
}

func (r *PartyRegistry) disbandForEmpty(partyID UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.disbandForEmptyInternal(partyID)
}

func (r *PartyRegistry) Disband(partyID UUID, sender UUID) (bool, string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	party := r.getPartyInternal(partyID)
	if party == nil {
		return false, "ERR_INVALID_PARTY"
	}
	if party.CurrentLeader != sender {
		return false, "ERR_NOT_LEADER"
	}

	msg, _ := json.Marshal(&PartyOnePlayerPacket{
		PartyID:  partyID,
		PlayerID: sender,
	})
	err := r.nc.Publish(env.EnsurePrefixed("party.disband.notify.command"), msg)
	if err != nil {
		return false, "ERR_BROADCAST_FAILED"
	}

	delete(r.parties, partyID)

	log.Printf("Party %s has been disbanded by %s", partyID, sender)
	return true, ""
}

func (r *PartyRegistry) JoinParty(partyID UUID, player UUID, fromInvite bool) (success bool, error string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	//todo: consider performing online checks on this player
	// It would need to involve a redis check
	if !r.containsKeyInternal(partyID) {
		log.Printf("Failed to join party: registry does not contain party: %s", partyID)
		return
	}

	if r.isInPartyInternal(player) {
		return false, "ERR_ALREADY_IN_PARTY"
	}

	party := r.parties[partyID]
	if !fromInvite && !party.Open {
		return false, "ERR_NO_INVITE"
	}

	if fromInvite {
		// remove invite
		var toRemove *UUID = nil
		for _, invite := range party.ActiveInvites {
			if invite.Recipient == player {
				toRemove = &invite.ID
			}
		}
		if toRemove != nil { // this is used for bypass, so may be nil
			invites := party.ActiveInvites
			delete(invites, *toRemove)
			party.ActiveInvites = invites
		}
	}

	msg, _ := json.Marshal(&PartyOnePlayerPacket{
		PartyID:  partyID,
		PlayerID: player,
	})
	err := r.nc.Publish(env.EnsurePrefixed("party.join.notify"), msg)
	if err != nil {
		return false, "ERR_BROADCAST_FAILED"
	}

	party.Members = append(party.Members, player)
	r.parties[partyID] = party
	log.Printf("PlayerID %s joined party %s", player, partyID)

	return true, ""
}

func (r *PartyRegistry) DisconnectFromParty(player UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.isInPartyInternal(player) {
		return
	}
	_, party := r.getPlayerPartyInternal(player)

	delete(r.disconnects, player)

	if party.TotalSize() <= 2 {
		// if a party has 2 members, it will be empty once the player is removed.
		r.disbandForEmptyInternal(party.ID)
		return
	}

	if party.IsMember(player) {
		party.Members = removeUUID(party.Members, player)
	} else if party.IsModerator(player) {
		party.Moderators = removeUUID(party.Moderators, player)
	} else {
		party.CurrentLeader = r.selectNewLeader(*party)
		r.parties[party.ID] = *party

		msg1, _ := json.Marshal(&PartyTwoPlayerPacket{
			PartyID:  party.ID,
			SenderID: player,
			PlayerID: party.CurrentLeader,
		})
		_ = r.nc.Publish(env.EnsurePrefixed("party.transfer.disconnected"), msg1)
		return
	}

	msg, _ := json.Marshal(&PartyOnePlayerPacket{
		PartyID:  party.ID,
		PlayerID: player,
	})
	err := r.nc.Publish(env.EnsurePrefixed("party.leave.notify.disconnected"), msg)
	if err != nil {
		log.Printf("Failed to broadcast party remove disconnected: %v", err)
		return
	}

	r.parties[party.ID] = *party
	log.Printf("PlayerID %s disconnected from party %s", player, party.ID)
}

func (r *PartyRegistry) LeaveParty(player UUID) (success bool, error string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.isInPartyInternal(player) {
		return false, "ERR_NOT_IN_PARTY"
	}

	_, party := r.getPlayerPartyInternal(player)

	if party.TotalSize() <= 2 {
		r.disbandForEmptyInternal(party.ID)
		return true, ""
	}

	if party.CurrentLeader == player {
		party.Moderators = removeUUID(party.Moderators, player)
		party.Members = removeUUID(party.Members, player)

		newLeader := r.selectNewLeader(*party)
		if newLeader == UUID(uuid.Nil) {
			// no new leader, disband party
			r.disbandForEmptyInternal(party.ID)
			return true, ""
		}

		party.CurrentLeader = newLeader
		r.parties[party.ID] = *party

		msg, _ := json.Marshal(&PartyTwoPlayerPacket{
			PartyID:  party.ID,
			SenderID: player,
			PlayerID: newLeader,
		})
		err := r.nc.Publish(env.EnsurePrefixed("party.transfer.left"), msg)
		if err != nil {
			return false, "ERR_BROADCAST_FAILED"
		}
		return true, ""
	}

	msg, _ := json.Marshal(&PartyOnePlayerPacket{
		PartyID:  party.ID,
		PlayerID: player,
	})
	err := r.nc.Publish(env.EnsurePrefixed("party.leave.notify.command"), msg)
	if err != nil {
		return false, "ERR_BROADCAST_FAILED"
	}

	party.Members = removeUUID(party.Members, player)
	r.parties[party.ID] = *party
	log.Printf("PlayerID %s left party %s", player, party.ID)

	return true, ""
}

func (r *PartyRegistry) Promote(sender UUID, partyID UUID, player UUID) (success bool, error string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	party := r.getPartyInternal(partyID)
	if party == nil {
		return false, "ERR_INVALID_PARTY"
	}
	if party.CurrentLeader != sender {
		return false, "ERR_NOT_LEADER"
	}
	if !party.IsInParty(player) {
		return false, "ERR_TARGET_NOT_IN_PARTY"
	}
	msg, _ := json.Marshal(&PartyTwoPlayerPacket{
		PartyID:  partyID,
		PlayerID: player,
		SenderID: sender,
	})
	if party.IsMember(player) {
		party.Moderators = append(party.Moderators, player)
		party.Members = removeUUID(party.Members, player)

		err := r.nc.Publish(env.EnsurePrefixed("party.promote.notify.moderator"), msg)
		if err != nil {
			return false, "ERR_BROADCAST_FAILED"
		}
		return true, ""
	}
	if party.IsModerator(player) {
		currentLeader := party.CurrentLeader
		party.CurrentLeader = player
		party.Moderators = append(removeUUID(party.Moderators, player), currentLeader)

		err := r.nc.Publish(env.EnsurePrefixed("party.promote.notify.leader"), msg)
		if err != nil {
			return false, "ERR_BROADCAST_FAILED"
		}
		return true, ""
	}
	return false, "ERR_ALREADY_LEADER"
}

func (r *PartyRegistry) Kick(sender UUID, partyID UUID, player UUID) (success bool, error string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	party := r.getPartyInternal(partyID)
	if party == nil {
		return false, "ERR_INVALID_PARTY"
	}
	if party.CurrentLeader != sender && !party.IsModerator(sender) {
		return false, "ERR_NO_KICK_PERMISSION"
	}
	if player == party.CurrentLeader {
		return false, "ERR_CANNOT_KICK_LEADER"
	}
	if player == sender {
		return false, "ERR_CANNOT_KICK_SELF" // just leave lol
	}
	if !party.IsInParty(player) {
		return false, "ERR_TARGET_NOT_IN_PARTY"
	}

	msg, _ := json.Marshal(&PartyTwoPlayerPacket{
		PartyID:  partyID,
		PlayerID: player,
		SenderID: sender,
	})
	_ = r.nc.Publish(env.EnsurePrefixed("party.kick.notify"), msg)

	party.Members = removeUUID(party.Members, player)
	party.Moderators = removeUUID(party.Moderators, player)
	r.parties[partyID] = *party

	if party.TotalSize() <= 1 {
		r.disbandForEmptyInternal(partyID)
	}

	return true, ""
}

func (r *PartyRegistry) Transfer(sender UUID, partyID UUID, player UUID) (success bool, error string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	party := r.getPartyInternal(partyID)
	if party == nil {
		return false, "ERR_INVALID_PARTY"
	}
	if !party.IsInParty(player) {
		return false, "ERR_TARGET_NOT_IN_PARTY"
	}
	if party.CurrentLeader != sender {
		return false, "ERR_NOT_LEADER"
	}
	party.Moderators = append(removeUUID(party.Moderators, player), sender)
	party.Members = removeUUID(party.Members, player)
	party.CurrentLeader = player
	r.parties[partyID] = *party

	msg, _ := json.Marshal(&PartyTwoPlayerPacket{
		PartyID:  partyID,
		PlayerID: player,
		SenderID: sender,
	})
	_ = r.nc.Publish(env.EnsurePrefixed("party.transfer.command"), msg)

	return true, ""
}

func (r *PartyRegistry) ToggleMute(sender UUID, partyID UUID, state bool) (success bool, error string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	party := r.getPartyInternal(partyID)
	if party == nil {
		return false, "ERR_INVALID_PARTY"
	}
	if party.CurrentLeader != sender {
		return false, "ERR_NO_PERMISSION"
	}

	party.Muted = state
	r.parties[partyID] = *party

	msg, _ := json.Marshal(&PartyStateChangePacket{
		PartyID:  partyID,
		PlayerID: sender,
		State:    state,
	})
	_ = r.nc.Publish(env.EnsurePrefixed("party.state.mute.notify"), msg)
	return true, ""
}

func (r *PartyRegistry) Yoink(sender UUID, partyID UUID) (success bool, error string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	party := r.getPartyInternal(partyID)
	if party == nil {
		return false, "ERR_INVALID_PARTY"
	}
	if party.CurrentLeader == sender {
		return false, "ERR_ALREADY_LEADER"
	}
	party.Moderators = append(removeUUID(party.Moderators, sender), party.CurrentLeader)
	party.Members = removeUUID(party.Members, sender)
	party.CurrentLeader = sender
	r.parties[partyID] = *party

	msg, _ := json.Marshal(&PartyOnePlayerPacket{
		PartyID:  partyID,
		PlayerID: sender,
	})
	_ = r.nc.Publish(env.EnsurePrefixed("party.yoink.notify"), msg)

	return true, ""
}

func (r *PartyRegistry) ToggleOpenInvites(sender UUID, partyID UUID, state bool) (success bool, error string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	party := r.getPartyInternal(partyID)
	if party == nil {
		return false, "ERR_INVALID_PARTY"
	}
	if party.CurrentLeader != sender {
		return false, "ERR_NO_PERMISSION"
	}

	party.OpenInvites = state
	r.parties[partyID] = *party

	msg, _ := json.Marshal(&PartyStateChangePacket{
		PartyID:  partyID,
		PlayerID: sender,
		State:    state,
	})
	_ = r.nc.Publish(env.EnsurePrefixed("party.state.open_invites.notify"), msg)
	return true, ""
}

func (r *PartyRegistry) ToggleOpen(sender UUID, partyID UUID, state bool) (success bool, error string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	party := r.getPartyInternal(partyID)
	if party == nil {
		return false, "ERR_INVALID_PARTY"
	}
	if party.CurrentLeader != sender {
		return false, "ERR_NO_PERMISSION"
	}

	party.Open = state
	r.parties[partyID] = *party

	msg, _ := json.Marshal(&PartyStateChangePacket{
		PartyID:  partyID,
		PlayerID: sender,
		State:    state,
	})
	_ = r.nc.Publish(env.EnsurePrefixed("party.state.open.notify"), msg)
	return true, ""
}

func (r *PartyRegistry) selectNewLeader(party Party) UUID {
	if len(party.Moderators) != 0 {
		return party.Moderators[0]
	}
	if len(party.Members) != 0 {
		return party.Members[0]
	}
	return UUID(uuid.Nil)
}

func (r *PartyRegistry) GetParty(id UUID) *Party {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.getPartyInternal(id)
}

func (r *PartyRegistry) getPartyInternal(id UUID) *Party {
	party, ok := r.parties[id]
	if !ok {
		return nil
	}
	return &party
}

func (r *PartyRegistry) GetAllParties() []Party {
	r.mu.Lock()
	defer r.mu.Unlock()
	values := make([]Party, 0, len(r.parties))

	for _, v := range r.parties {
		values = append(values, v)
	}
	return values
}

func (r *PartyRegistry) contains(invite Party) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, party := range r.parties {
		if invite.ID == party.ID {
			return true
		}
	}
	return false
}

// IsInParty checks if the player identified by the given UUID is a member of any party in the registry.
func (r *PartyRegistry) IsInParty(player UUID) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.isInPartyInternal(player)
}

func (r *PartyRegistry) isInPartyInternal(player UUID) bool {
	for _, party := range r.parties {
		if party.IsInParty(player) {
			return true
		}
	}
	return false
}

func (r *PartyRegistry) GetPlayerParty(playerID UUID) (bool, *Party) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.getPlayerPartyInternal(playerID)
}

func (r *PartyRegistry) getPlayerPartyInternal(playerID UUID) (bool, *Party) {
	if !r.isInPartyInternal(playerID) {
		return false, nil
	}
	for _, party := range r.parties {
		if party.IsInParty(playerID) {
			return true, &party
		}
	}
	return false, nil // will never happen
}

func (r *PartyRegistry) containsKey(id UUID) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.containsKeyInternal(id)
}

func (r *PartyRegistry) containsKeyInternal(id UUID) bool {
	for u := range r.parties {
		if id == u {
			return true
		}
	}
	return false
}

func removeUUID(slice []UUID, s UUID) []UUID {
	for i, v := range slice {
		if v == s {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

func (r *PartyRegistry) HandleDisconnect(playerID UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.isInPartyInternal(playerID) {
		return
	}

	_, party := r.getPlayerPartyInternal(playerID)

	ctx, cancel := context.WithCancel(context.Background())
	r.disconnects[playerID] = cancel

	// Start removal timer in goroutine
	go func() {
		select {
		case <-time.After(5 * time.Minute):
			r.DisconnectFromParty(playerID)
		case <-ctx.Done():
			fmt.Printf("Removal cancelled for player %s\n", playerID)
			return
		}
	}()

	msg, _ := json.Marshal(&PartyOnePlayerPacket{
		PlayerID: playerID,
		PartyID:  party.ID,
	})
	err := r.nc.Publish(env.EnsurePrefixed("party.status.disconnect"), msg)
	if err != nil {
		log.Printf("Failed to broadcast party remove disconnected: %v", err)
	}

	log.Printf("PlayerID %s disconnected, will be removed in 5 minutes\n", playerID)
}

func (r *PartyRegistry) HandleReconnect(playerID UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Cancel the removal timer if it exists
	if cancel, exists := r.disconnects[playerID]; exists {
		cancel()
		delete(r.disconnects, playerID)
		log.Printf("PlayerID %s reconnected, removal cancelled\n", playerID)
	}

	if !r.isInPartyInternal(playerID) {
		return
	}

	_, party := r.getPlayerPartyInternal(playerID)
	msg, _ := json.Marshal(&PartyOnePlayerPacket{
		PartyID:  party.ID,
		PlayerID: playerID,
	})
	_ = r.nc.Publish(env.EnsurePrefixed("party.status.reconnect"), msg)
}
