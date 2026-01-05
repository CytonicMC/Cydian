package handlers

import (
	"encoding/json"
	"log"
	"strings"

	"github.com/CytonicMC/Cydian/env"
	"github.com/CytonicMC/Cydian/parties"
	"github.com/nats-io/nats.go"
)

func RegisterParties(nc *nats.Conn, registry *parties.PartyRegistry) {
	disbandHandler(nc, registry)
	joinHandler(nc, registry)
	leaveHandler(nc, registry)
	promoteHandler(nc, registry)
	transferHandler(nc, registry)
	kickHandler(nc, registry)
	stateHandler(nc, registry)
	fetchHandler(nc, registry)
	yoinkHandler(nc, registry)
}

func disbandHandler(nc *nats.Conn, registry *parties.PartyRegistry) {
	const subject = "party.disband.request"

	_, err := nc.Subscribe(env.EnsurePrefixed(subject), func(msg *nats.Msg) {
		var packet parties.PartyOnePlayerPacket
		if err := json.Unmarshal(msg.Data, &packet); err != nil {
			log.Printf("Invalid PartyOnePlayerPacket message format: %s", msg.Data)
			return
		}

		success, reason := registry.Disband(packet.PartyID, packet.PlayerID)
		reply(msg, success, reason)
	})
	if err != nil {
		log.Fatalf("Error subscribing to subject %s: %v", subject, err)
	}
	log.Printf("Listening for party disbands on subject '%s'", subject)
}

func joinHandler(nc *nats.Conn, registry *parties.PartyRegistry) {
	const subject = "party.join.request.*" // allow for bypass using wildcard

	_, err := nc.Subscribe(env.EnsurePrefixed(subject), func(msg *nats.Msg) {
		var packet parties.PartyOnePlayerPacket
		if err := json.Unmarshal(msg.Data, &packet); err != nil {
			log.Printf("Invalid PartyOnePlayerPacket message format: %s", msg.Data)
			return
		}
		party := registry.GetParty(packet.PartyID)
		if party == nil {
			ack, _ := json.Marshal(&parties.GenericPartyResponsePacket{
				Success: false,
				Message: "INVALID_PARTY",
			})
			err := msg.Respond(ack)
			if err != nil {
				log.Printf("Error sending acknowledgment: %v", err)
			}
			return
		}
		hasBypassed := strings.Contains(msg.Subject, "bypass")
		success, reason := registry.JoinParty(party.ID, packet.PlayerID, hasBypassed)
		reply(msg, success, reason)
	})
	if err != nil {
		log.Fatalf("Error subscribing to subject %s: %v", subject, err)
	}
	log.Printf("Listening for party join requests on subject '%s'", subject)
}

func leaveHandler(nc *nats.Conn, registry *parties.PartyRegistry) {
	const subject = "party.leave.request"

	_, err := nc.Subscribe(env.EnsurePrefixed(subject), func(msg *nats.Msg) {
		var packet parties.PartyLeaveRequestPacket
		if err := json.Unmarshal(msg.Data, &packet); err != nil {
			log.Printf("Invalid PartyLeaveRequestPacket message format: %s", msg.Data)
			return
		}

		success, reason := registry.LeaveParty(packet.PlayerID)
		reply(msg, success, reason)
	})
	if err != nil {
		log.Fatalf("Error subscribing to subject %s: %v", subject, err)
	}
	log.Printf("Listening for party invite acceptances on subject '%s'", subject)
}

func promoteHandler(nc *nats.Conn, registry *parties.PartyRegistry) {
	const subject = "party.promote.request"

	_, err := nc.Subscribe(env.EnsurePrefixed(subject), func(msg *nats.Msg) {
		var packet parties.PartyTwoPlayerPacket
		if err := json.Unmarshal(msg.Data, &packet); err != nil {
			log.Printf("Invalid PartyTwoPlayerPacket message format: %s", msg.Data)
			return
		}

		success, reason := registry.Promote(packet.SenderID, packet.PartyID, packet.PlayerID)
		reply(msg, success, reason)
	})
	if err != nil {
		log.Fatalf("Error subscribing to subject %s: %v", subject, err)
	}
	log.Printf("Listening for party promotions on subject '%s'", subject)
}

func transferHandler(nc *nats.Conn, registry *parties.PartyRegistry) {
	const subject = "party.transfer.request"

	_, err := nc.Subscribe(env.EnsurePrefixed(subject), func(msg *nats.Msg) {
		var packet parties.PartyTwoPlayerPacket
		if err := json.Unmarshal(msg.Data, &packet); err != nil {
			log.Printf("Invalid PartyTransferPacket message format: %s", msg.Data)
			return
		}

		success, reason := registry.Transfer(packet.SenderID, packet.PartyID, packet.PlayerID)
		reply(msg, success, reason)
	})
	if err != nil {
		log.Fatalf("Error subscribing to subject %s: %v", subject, err)
	}
	log.Printf("Listening for party transfers on subject '%s'", subject)
}

func yoinkHandler(nc *nats.Conn, registry *parties.PartyRegistry) {
	const subject = "party.yoink.request"

	_, err := nc.Subscribe(env.EnsurePrefixed(subject), func(msg *nats.Msg) {
		var packet parties.PartyOnePlayerPacket
		if err := json.Unmarshal(msg.Data, &packet); err != nil {
			log.Printf("Invalid PartyOnePlayerPacket message format: %s", msg.Data)
			return
		}

		success, reason := registry.Yoink(packet.PlayerID, packet.PartyID)
		reply(msg, success, reason)
	})
	if err != nil {
		log.Fatalf("Error subscribing to subject %s: %v", subject, err)
	}
	log.Printf("Listening for party yoinks on subject '%s'", subject)
}

func kickHandler(nc *nats.Conn, registry *parties.PartyRegistry) {
	const subject = "party.kick.request"

	_, err := nc.Subscribe(env.EnsurePrefixed(subject), func(msg *nats.Msg) {
		var packet parties.PartyTwoPlayerPacket
		if err := json.Unmarshal(msg.Data, &packet); err != nil {
			log.Printf("Invalid PartyTwoPlayerPacket message format: %s", msg.Data)
			return
		}

		success, reason := registry.Kick(packet.SenderID, packet.PartyID, packet.PlayerID)
		reply(msg, success, reason)
	})
	if err != nil {
		log.Fatalf("Error subscribing to subject %s: %v", subject, err)
	}
	log.Printf("Listening for party kicks on subject '%s'", subject)
}

func stateHandler(nc *nats.Conn, registry *parties.PartyRegistry) {
	const subject = "party.state.*.request"

	_, err := nc.Subscribe(env.EnsurePrefixed(subject), func(msg *nats.Msg) {
		var packet parties.PartyStateChangePacket
		if err := json.Unmarshal(msg.Data, &packet); err != nil {
			log.Printf("Invalid PartyStateChangePacket message format: %s", msg.Data)
			return
		}

		parts := strings.Split(msg.Subject, ".")

		// party.state.<action>.request
		if len(parts) != 4 {
			return
		}

		action := parts[2]

		var success bool
		var reason string

		switch action {
		case "mute":
			success, reason = registry.ToggleMute(packet.PlayerID, packet.PartyID, packet.State)
			break
		case "open_invites":
			success, reason = registry.ToggleOpenInvites(packet.PlayerID, packet.PartyID, packet.State)
			break
		case "open":
			success, reason = registry.ToggleOpen(packet.PlayerID, packet.PartyID, packet.State)
			break
		default:
			success = false
			reason = "ERR_INVALID_ACTION"
			log.Printf("Invalid party state change action: %s", action)
			return
		}
		reply(msg, success, reason)
	})
	if err != nil {
		log.Fatalf("Error subscribing to subject %s: %v", subject, err)
	}
	log.Printf("Listening for party state changes on subject '%s'", subject)
}

func fetchHandler(nc *nats.Conn, registry *parties.PartyRegistry) {
	const subject = "party.fetch.request"

	_, err := nc.Subscribe(env.EnsurePrefixed(subject), func(msg *nats.Msg) {

		data := registry.GetAllParties()
		ack, err := json.Marshal(&data)
		if err != nil {
			log.Printf("Error marshalling fetch party packet response: %v", err)
			return
		}
		err1 := msg.Respond(ack)
		if err1 != nil {
			log.Printf("Error sending acknowledgment: %v", err1)
		}
	})
	if err != nil {
		log.Fatalf("Error subscribing to subject %s: %v", subject, err)
	}
	log.Printf("Listening for party fetch requests on subject '%s'", subject)
}

func reply(msg *nats.Msg, success bool, reason string) {
	ack, err1 := json.Marshal(&parties.GenericPartyResponsePacket{
		Success: success,
		Message: reason,
	})
	if err1 != nil {
		log.Printf("Error marshalling party packet response: %v", err1)
		return
	}
	if err := msg.Respond(ack); err != nil {
		log.Printf("Error sending acknowledgment: %v", err)
	}
}
