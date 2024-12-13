package handlers

import (
	"encoding/json"
	"github.com/CytonicMC/Cydian/friends"
	"github.com/nats-io/nats.go"
	"log"
)

func RegisterFriends(nc *nats.Conn, registry *friends.Registry) {
	acceptHandler(nc, registry)
	declineHandler(nc, registry)
}

func acceptHandler(nc *nats.Conn, registry *friends.Registry) {
	const subject = "friends.accept"

	_, err := nc.Subscribe(subject, func(msg *nats.Msg) {
		var packet friends.FriendResponse
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
			sendAcceptance(nc, req)
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
	log.Printf("Listening for friend acceptances on subject '%s'", subject)
}

func declineHandler(nc *nats.Conn, registry *friends.Registry) {
	const subject = "friends.decline"

	_, err := nc.Subscribe(subject, func(msg *nats.Msg) {
		var packet friends.FriendResponse
		if err := json.Unmarshal(msg.Data, &packet); err != nil {
			log.Printf("Invalid message format: %s", msg.Data)
			return
		}

		success, req := registry.Decline(packet.ID)
		// Add or update the server in the registry
		if success {
			ack := []byte("SUCCESS")
			if err := msg.Respond(ack); err != nil {
				log.Printf("Error sending acknowledgment: %v", err)
			}
			sendDeclination(nc, req)
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
	log.Printf("Listening for friend declinations on subject '%s'", subject)
}

func sendAcceptance(nc *nats.Conn, req friends.FriendRequest) {

	const subject = "friends.accept.notify"
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

func sendDeclination(nc *nats.Conn, req friends.FriendRequest) {

	const subject = "friends.decline.notify"
	marshal, errr := json.Marshal(req)
	if errr != nil {
		log.Fatalf("Error marshalling friend request: %v", errr)
		return
	}
	err := nc.Publish(subject, marshal)
	if err != nil {
		log.Fatalf("Error publishing friends declination message: %v", err)
		return
	}
}
