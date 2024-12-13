package handlers

import (
	"encoding/json"
	"github.com/CytonicMC/Cydian/parties"
	"github.com/nats-io/nats.go"
	"log"
)

func RegisterPartyInvites(nc *nats.Conn, registry *parties.InviteRegistry) {
	acceptInviteHandler(nc, registry)
}

// on accept
func acceptInviteHandler(nc *nats.Conn, registry *parties.InviteRegistry) {
	const subject = "party.invites.accept"

	_, err := nc.Subscribe(subject, func(msg *nats.Msg) {
		var packet parties.PartyInviteAccept
		if err := json.Unmarshal(msg.Data, &packet); err != nil {
			log.Printf("Invalid message format: %s", msg.Data)
			return
		}

		success, req := registry.Accept(packet.ID)

		// Add or update the server in the registry
		if success {
			ack := []byte("SUCCESS")
			if err := msg.Respond(ack); err != nil {
				log.Printf("Error sending acknowledgment: %v", err)
			}
			sendJoin(nc, req)
		} else {
			ack := []byte("FAILURE")
			if err := msg.Respond(ack); err != nil {
				log.Printf("Error sending acknowledgment: %v", err)
			}
		}
	})
	if err != nil {
		log.Fatalf("Error subscribing to subject %s: %v", subject, err)
	}
	log.Printf("Listening for party invite acceptances on subject '%s'", subject)
}

func sendJoin(nc *nats.Conn, req parties.PartyInvite) {

	const subject = "party.invites.accept.notify"
	marshal, errr := json.Marshal(req)
	if errr != nil {
		log.Fatalf("Error marshalling friend request: %v", errr)
		return
	}
	err := nc.Publish(subject, marshal)
	if err != nil {
		log.Fatalf("Error publishing friends acceptance message: %v", err)
		return
	}
}
