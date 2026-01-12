package parties

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type UUID uuid.UUID

func (u *UUID) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	parsed, err := uuid.Parse(s)
	if err != nil {
		return err
	}
	*u = UUID(parsed)
	return nil
}

func (u UUID) MarshalJSON() ([]byte, error) {
	return json.Marshal(uuid.UUID(u).String())
}

func (u UUID) MarshalText() ([]byte, error) {
	return []byte(uuid.UUID(u).String()), nil
}

func (u *UUID) UnmarshalText(data []byte) error {
	parsed, err := uuid.Parse(string(data))
	if err != nil {
		return err
	}
	*u = UUID(parsed)
	return nil
}

type Party struct {
	ID            UUID                 `json:"id"`
	CurrentLeader UUID                 `json:"current_leader"` // can 100% change, don't use it as a key. They are NOT included in the Members field
	Moderators    *Set                 `json:"moderators"`     // they are NOT included in Members, just like CurrentLeader
	Members       *Set                 `json:"members"`        // all "standard" members of the party. The leader and moderators are NOT part of this list
	Open          bool                 `json:"open"`           // anyone can join it with /p join <any member's name>
	OpenInvites   bool                 `json:"open_invites"`
	Muted         bool                 `json:"muted"`          // no one can speak except for moderators
	ActiveInvites map[UUID]PartyInvite `json:"active_invites"` // keyed by invite uuid
}

type Set struct {
	elements map[UUID]struct{}
}

// NewSet creates a new set
func NewSet() *Set {
	return &Set{
		elements: make(map[UUID]struct{}),
	}
}

func NewSetFromSlice(slice []UUID) *Set {
	set := NewSet()
	for _, v := range slice {
		set.Add(v)
	}
	return set
}

// Add inserts an element into the set
func (s *Set) Add(value UUID) {
	s.elements[value] = struct{}{}
}

// Remove deletes an element from the set
func (s *Set) Remove(value UUID) {
	delete(s.elements, value)
}

// Contains checks if an element is in the set
func (s *Set) Contains(value UUID) bool {
	_, found := s.elements[value]
	return found
}

// Size returns the number of elements in the set
func (s *Set) Size() int {
	return len(s.elements)
}

// Slice returns all elements in the set as a slice
func (s *Set) Slice() []UUID {
	keys := make([]UUID, 0, len(s.elements))
	for key := range s.elements {
		keys = append(keys, key)
	}
	return keys
}

func (s *Set) UnmarshalJSON(data []byte) error {
	var slice []UUID
	if err := json.Unmarshal(data, &slice); err != nil {
		return err
	}

	*s = *NewSetFromSlice(slice)
	return nil
}

func (s Set) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.Slice())
}

func (p Party) IsInParty(playerID UUID) bool {
	// Check leader first
	if p.CurrentLeader == playerID {
		return true
	}

	// Check moderators
	if p.Moderators.Contains(playerID) {
		return true
	}

	// Check members
	if p.Members.Contains(playerID) {
		return true
	}

	return false
}

func (p Party) IsMember(playerID UUID) bool {
	return p.Members.Contains(playerID)
}

func (p Party) IsModerator(playerID UUID) bool {
	return p.Moderators.Contains(playerID)
}

func (p Party) TotalSize() int {
	return p.Members.Size() + p.Moderators.Size() + len(p.ActiveInvites) + 1
}

type PartyInvite struct {
	ID        UUID      `json:"id"`
	PartyID   UUID      `json:"party_id"`
	Recipient UUID      `json:"recipient"`
	SenderID  UUID      `json:"sender_id"` // can be a moderator or anyone if OpenInvites is enabled
	Expiry    time.Time `json:"expiry"`
}

type PartyInviteSendPacket struct {
	PartyID     *UUID `json:"party_id"` // this may be nil
	SenderID    UUID  `json:"sender_id"`
	RecipientID UUID  `json:"recipient_id"`
}

type PartyInviteAcceptPacket struct {
	RequestID UUID `json:"request_id"`
}

type PartyInviteExpirePacket struct {
	RequestID UUID `json:"request_id"`
	PartyID   UUID `json:"party_id"`
	Recipient UUID `json:"recipient"`
	SenderID  UUID `json:"sender_id"`
}

type PartyInvitePacket struct {
	Invite PartyInvite `json:"invite"`
}

type PartyLeaveRequestPacket struct {
	PlayerID UUID `json:"player_id"`
}

type PartyEmptyDisbandPacket struct {
	PartyID UUID `json:"party_id"`
}

type GenericPartyResponsePacket struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type PartyOnePlayerPacket struct {
	PartyID  UUID `json:"party_id"`
	PlayerID UUID `json:"player_id"`
}

type PartyTwoPlayerPacket struct {
	PartyID  UUID `json:"party_id"`
	PlayerID UUID `json:"player_id"`
	SenderID UUID `json:"sender_id"`
}

type PartyStateChangePacket struct {
	PartyID  UUID `json:"party_id"`
	PlayerID UUID `json:"player_id"`
	State    bool `json:"state"`
}

type PartyCreatePacket struct {
	Party Party `json:"party"`
}
