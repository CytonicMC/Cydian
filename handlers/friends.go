package handlers

import (
	"encoding/json"
	"github.com/CytonicMC/Cydian/friends"
	"github.com/nats-io/nats.go"
	"log"
)

func RegisterFriends(nc *nats.Conn, registry *friends.Registry) {
	acceptHandlerId(nc, registry)
	declineHandlerId(nc, registry)
	acceptHandler(nc, registry)
	declineHandler(nc, registry)
	requestHandler(nc, registry)
}

func acceptHandlerId(nc *nats.Conn, registry *friends.Registry) {
	const subject = "friends.accept.by_id"

	_, err := nc.Subscribe(subject, func(msg *nats.Msg) {
		var packet friends.FriendResponseId
		if err := json.Unmarshal(msg.Data, &packet); err != nil {
			log.Printf("Invalid message format: %s", msg.Data)
			return
		}

		success, req := registry.AcceptByID(packet.ID)

		// Add or update the server in the registry
		if success {
			ack, _ := json.Marshal(friends.FriendRequestApiResponse{
				Success: true,
				Code:    "SUCCESS",
				Message: "Request successfully accepted",
			})
			if err := msg.Respond(ack); err != nil {
				log.Printf("Error sending acknowledgment: %v", err)
			}
			friends.SendAcceptance(nc, req)
		} else {
			ack, _ := json.Marshal(friends.FriendRequestApiResponse{
				Success: false,
				Code:    "NOT_FOUND",
				Message: "No valid request to accept.",
			})
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

func declineHandlerId(nc *nats.Conn, registry *friends.Registry) {
	const subject = "friends.decline.by_id"

	_, err := nc.Subscribe(subject, func(msg *nats.Msg) {
		var packet friends.FriendResponseId
		if err := json.Unmarshal(msg.Data, &packet); err != nil {
			log.Printf("Invalid message format: %s", msg.Data)
			return
		}

		success, req := registry.DeclineByID(packet.ID)
		// Add or update the server in the registry
		if success {
			ack, _ := json.Marshal(friends.FriendRequestApiResponse{
				Success: true,
				Code:    "SUCCESS",
				Message: "Request successfully declined",
			})
			if err := msg.Respond(ack); err != nil {
				log.Printf("Error sending acknowledgment: %v", err)
			}
			sendDeclination(nc, req)
		} else {
			ack, _ := json.Marshal(friends.FriendRequestApiResponse{
				Success: false,
				Code:    "NOT_FOUND",
				Message: "No valid request to decline.",
			})
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

func acceptHandler(nc *nats.Conn, registry *friends.Registry) {
	const subject = "friends.accept"

	_, err := nc.Subscribe(subject, func(msg *nats.Msg) {
		var packet friends.FriendResponse
		if err := json.Unmarshal(msg.Data, &packet); err != nil {
			log.Printf("Invalid message format: %s", msg.Data)
			return
		}

		success, req := registry.Accept(packet.Sender, packet.Recipient)

		// Add or update the server in the registry
		if success {
			ack, _ := json.Marshal(friends.FriendRequestApiResponse{
				Success: true,
				Code:    "SUCCESS",
				Message: "Request successfully accepted",
			})
			if err := msg.Respond(ack); err != nil {
				log.Printf("Error sending acknowledgment: %v", err)
			}
			friends.SendAcceptance(nc, req)
		} else {
			ack, _ := json.Marshal(friends.FriendRequestApiResponse{
				Success: false,
				Code:    "NOT_FOUND",
				Message: "No valid request to accept.",
			})
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

		success, req := registry.Decline(packet.Sender, packet.Recipient)
		// Add or update the server in the registry
		if success {
			ack, _ := json.Marshal(friends.FriendRequestApiResponse{
				Success: true,
				Code:    "SUCCESS",
				Message: "Request successfully declined",
			})
			if err := msg.Respond(ack); err != nil {
				log.Printf("Error sending acknowledgment: %v", err)
			}
			sendDeclination(nc, req)
		} else {
			ack, _ := json.Marshal(friends.FriendRequestApiResponse{
				Success: false,
				Code:    "NOT_FOUND",
				Message: "No valid request to decline.",
			})
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

func requestHandler(nc *nats.Conn, req *friends.Registry) {
	const subject = "friends.request"

	_, err := nc.Subscribe(subject, func(msg *nats.Msg) {

		var packet friends.FriendRequest
		if err := json.Unmarshal(msg.Data, &packet); err != nil {
			log.Printf("Invalid message format: %s", msg.Data)
			return
		}

		// register the request
		success, dontSend := req.AddOrUpdate(packet)
		if !success {
			log.Printf("ALREADY_SENT")

			ack, _ := json.Marshal(friends.FriendRequestApiResponse{
				Success: false,
				Code:    "ALREADY_SENT",
				Message: "You have already send a request to this player.",
			})
			if err := msg.Respond(ack); err != nil {
				log.Printf("Error sending acknowledgment: %v", err)
			}
		} else {
			log.Printf("Successfully sent")
			ack, _ := json.Marshal(friends.FriendRequestApiResponse{
				Success: true,
				Code:    "SUCCESS",
				Message: "Request successfully sent.",
			})
			if err := msg.Respond(ack); err != nil {
				log.Printf("Error sending acknowledgment: %v", err)
			}

			if !dontSend {
				err1 := nc.Publish("friends.request.notify", msg.Data)
				if err1 != nil {
					log.Printf("Error publishing friends request: %v", err1)
					return
				}
			}
		}

	})
	if err != nil {
		log.Fatalf("Error subscribing to subject %s: %v", subject, err)
	}
	log.Printf("Listening for friend requests on subject '%s'", subject)
}
