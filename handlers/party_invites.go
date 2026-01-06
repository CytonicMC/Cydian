package handlers

import (
	"encoding/json"
	"log"

	"github.com/CytonicMC/Cydian/app"
	"github.com/CytonicMC/Cydian/env"
	"github.com/CytonicMC/Cydian/parties"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

func RegisterPartyInvites(nc *nats.Conn, registry *parties.InviteRegistry, cydian *app.Cydian) {
	acceptInviteHandler(nc, registry)
	sendInviteHandler(nc, registry, cydian)
}

// on accept
func acceptInviteHandler(nc *nats.Conn, registry *parties.InviteRegistry) {
	const subject = "party.invites.accept"

	_, err := nc.Subscribe(env.EnsurePrefixed(subject), func(msg *nats.Msg) {
		var packet parties.PartyInviteAcceptPacket
		if err := json.Unmarshal(msg.Data, &packet); err != nil {
			log.Printf("Invalid PartyInviteAcceptPacket message format: %s", msg.Data)
			return
		}

		success, req := registry.Accept(packet.RequestID)

		// Add or update the server in the registry
		if success {

			ack, err1 := json.Marshal(&parties.GenericPartyResponsePacket{
				Success: true,
				Message: "",
			})
			if err1 != nil {
				log.Printf("Error marshalling party invite response: %v", err1)
				return
			}
			if err := msg.Respond(ack); err != nil {
				log.Printf("Error sending acknowledgment: %v", err)
			}

			sendJoin(nc, *req)
		} else {
			ack, err1 := json.Marshal(&parties.GenericPartyResponsePacket{
				Success: false,
				Message: "ERR_INVALID_INVITE",
			})
			if err1 != nil {
				log.Printf("Error marshalling party invite response: %v", err1)
				return
			}
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

func sendInviteHandler(nc *nats.Conn, registry *parties.InviteRegistry, cydian *app.Cydian) {
	const subject = "party.invites.send"

	_, err := nc.Subscribe(env.EnsurePrefixed(subject), func(msg *nats.Msg) {
		var packet parties.PartyInviteSendPacket
		if err := json.Unmarshal(msg.Data, &packet); err != nil {
			log.Printf("Invalid PartyInviteSendPacket message format: %s", msg.Data)
			reply(msg, false, "ERR_INVALID_MESSAGE_FORMAT")
			return
		}

		isInParty := cydian.PartyRegistry.IsInParty(packet.SenderID)
		var isNewParty bool
		partyID := packet.PartyID
		if partyID == nil {
			if isInParty {
				sendReply(msg, &parties.GenericPartyResponsePacket{
					Success: false,
					Message: "ERR_STATE_MISMATCH_SERVER",
				})
				return
			}
			isNewParty = true
			pID := parties.UUID(uuid.New())
			partyID = &pID
		} else if cydian.PartyRegistry.GetParty(*partyID) == nil {
			sendReply(msg, &parties.GenericPartyResponsePacket{
				Success: false,
				Message: "ERR_STATE_MISMATCH_SERVICE",
			})
			return
		}

		invite, errMsg := registry.CreateInvite(packet.SenderID, *partyID, packet.RecipientID)

		if len(errMsg) > 0 {
			sendReply(msg, &parties.GenericPartyResponsePacket{
				Success: false,
				Message: errMsg,
			})
			return
		}

		if isNewParty {
			cydian.PartyRegistry.CreateParty(*partyID, packet.SenderID, invite)
		}

		serialized, err1 := json.Marshal(invite)
		if err1 != nil {
			log.Printf("Error marshalling party invite: %v", err1)
			sendReply(msg, &parties.GenericPartyResponsePacket{
				Success: false,
				Message: "ERR_MARSHAL_INVITE",
			})
			return
		}
		sendReply(msg, &parties.GenericPartyResponsePacket{
			Success: true,
			Message: string(serialized),
		})

		// broadcast the invite sent
		if !isNewParty {
			ack, _ := json.Marshal(&parties.PartyInvitePacket{
				Invite: *invite,
			})
			err := nc.Publish(env.EnsurePrefixed("party.invites.send.notify"), ack)
			if err != nil {
				log.Printf("Error sending error party invite send announcment: %v", errMsg)
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
	err := nc.Publish(env.EnsurePrefixed(subject), marshal)
	if err != nil {
		log.Fatalf("Error publishing friends acceptance message: %v", err)
		return
	}
}

func sendReply(msg *nats.Msg, data *parties.GenericPartyResponsePacket) {
	ack, err1 := json.Marshal(data)
	if err1 != nil {
		log.Printf("Error marshalling party invite response: %v", err1)
		return
	}
	err := msg.Respond(ack)
	if err != nil {
		log.Printf("Error sending error message: %v", err)
	}
}
