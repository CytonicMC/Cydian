package app

import (
	"github.com/CytonicMC/Cydian/friends"
	"github.com/CytonicMC/Cydian/parties"
	"github.com/CytonicMC/Cydian/servers"
)

type Cydian struct {
	ServerRegistry        *servers.Registry
	FriendRequestRegistry *friends.Registry
	PartyInviteRegistry   *parties.InviteRegistry
	PartyRegistry         *parties.PartyRegistry
}
