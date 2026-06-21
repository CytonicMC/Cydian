package app

import (
	"github.com/CytonicMC/Cydian/internal/friends"
	"github.com/CytonicMC/Cydian/internal/parties"
	"github.com/CytonicMC/Cydian/internal/servers"
)

type Cydian struct {
	ServerRegistry        *servers.Registry
	FriendRequestRegistry *friends.Registry
	PartyInviteRegistry   *parties.InviteRegistry
	PartyRegistry         *parties.PartyRegistry
}
