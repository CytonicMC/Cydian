package handlers

import (
	"encoding/json"
	"log"

	"github.com/CytonicMC/Cydian/app"
	"github.com/CytonicMC/Cydian/env"
	"github.com/CytonicMC/Cydian/parties"
	"github.com/nats-io/nats.go"
)

type PlayerStatusPacket struct {
	UUID     parties.UUID `json:"uuid"`
	Username string       `json:"username"`
}

// RegisterPlayerHandlers Register handlers for the various services that depend on player actions
func RegisterPlayerHandlers(nc *nats.Conn, instance *app.Cydian) {
	registerPlayerJoinHandler(nc, instance)
	registerPlayerLeaveHandler(nc, instance)
}

func registerPlayerJoinHandler(nc *nats.Conn, instance *app.Cydian) {
	_, err := nc.Subscribe(env.EnsurePrefixed("players.connect"), func(msg *nats.Msg) {
		obj := PlayerStatusPacket{}
		err := json.Unmarshal(msg.Data, &obj)
		if err != nil {
			log.Println("Error parsing player status packet in PlayerID Join Handler: ", err)
			return
		}
		instance.PartyRegistry.HandleReconnect(obj.UUID)
	})
	if err != nil {
		log.Fatalf("Error subscribing to subject %s: %v", "players.connect", err)
		return
	}
}

func registerPlayerLeaveHandler(nc *nats.Conn, instance *app.Cydian) {
	_, err := nc.Subscribe(env.EnsurePrefixed("players.disconnect"), func(msg *nats.Msg) {
		obj := PlayerStatusPacket{}
		err := json.Unmarshal(msg.Data, &obj)
		if err != nil {
			log.Println("Error parsing player status packet in PlayerID Leave Handler: ", err)
			return
		}
		instance.PartyRegistry.HandleDisconnect(obj.UUID)
	})
	if err != nil {
		log.Fatalf("Error subscribing to subject %s: %v", "players.disconnect", err)
		return
	}
}
