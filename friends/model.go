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

// FriendResponse The json "packet" sent on declination or acceptance
type FriendResponse struct {
	ID uuid.UUID `json:"request_id"`
}
