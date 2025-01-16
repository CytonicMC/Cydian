package friends

import (
	"github.com/google/uuid"
	"time"
)

// FriendRequest An object representing an active friend request
type FriendRequest struct {
	Sender    uuid.UUID `json:"sender"`
	Recipient uuid.UUID `json:"recipient"`
	Expiry    time.Time `json:"expiry"`
}

// FriendResponseId The json "packet" sent on declination or acceptance
type FriendResponseId struct {
	ID uuid.UUID `json:"request_id"`
}

// FriendResponse The json "packet" send on declination or acceptance, but it uses more human values. (Mostly from commands, making cytosis work easier)
type FriendResponse struct {
	Sender    uuid.UUID `json:"sender"`
	Recipient uuid.UUID `json:"recipient"`
}

type FriendRequestApiResponse struct {
	Success bool   `json:"success"`
	Code    string `json:"code"` //ie: "ALREADY_SENT"
	Message string `json:"message"`
}
