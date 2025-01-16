package parties

import (
	"github.com/google/uuid"
	"time"
)

type Party struct {
	ID            uuid.UUID     `json:"id"`
	CurrentLeader uuid.UUID     `json:"current_leader"` // can 100% change, don't use it as a key. They are NOT included in the Members field
	Moderators    []uuid.UUID   `json:"moderators"`     // they are NOT included in Members, just like CurrentLeader
	Members       []uuid.UUID   `json:"members"`        // all "standard" members of the party. The leader and moderators are NOT part of this list
	Open          bool          `json:"open"`           // anyone can join it with /p join <any member's name>
	OpenInvites   bool          `json:"open_invites"`
	Muted         bool          `json:"muted"` // no one can speak except for moderators
	ActiveInvites []PartyInvite `json:"active_invites"`
}

type PartyInvite struct {
	ID        uuid.UUID `json:"id"`
	PartyID   uuid.UUID `json:"party_id"`
	Recipient uuid.UUID `json:"recipient"`
	Sender    uuid.UUID `json:"sender"` // can be a moderator, or anyone if OpenInvites is enabled
	Expiry    time.Time `json:"expiry"`
}

type PartyInviteAccept struct {
	ID uuid.UUID `json:"id"`
}
