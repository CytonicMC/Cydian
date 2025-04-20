package handlers

import (
	"encoding/json"
	"github.com/CytonicMC/Cydian/instances"
	"github.com/hashicorp/nomad/api"
	"github.com/nats-io/nats.go"
	"log"
)

func RegisterInstances(nc *nats.Conn) {
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		log.Fatal(err)
	}
	createHandler(nc, client)
}

func createHandler(nc *nats.Conn, client *api.Client) {
	const subject = "servers.create"
	_, err := nc.Subscribe(subject, func(msg *nats.Msg) {
		var packet instances.InstanceCreateRequest
		if err := json.Unmarshal(msg.Data, &packet); err != nil {
			log.Printf("Invalid message format: %s", msg.Data)
			return
		}

		job, _, errJobs := client.Jobs().Info(packet.InstanceType, nil)
		if errJobs != nil {
			log.Printf("Error getting job info: %v", errJobs)
			reponse, _ := json.Marshal(instances.InstanceCreateResponse{
				Success: false,
				Message: "JOB_NOT_FOUND",
			})
			err := msg.Respond(reponse)
			if err != nil {
				log.Printf("Error sending acknowledgment: %v", err)
			}
			return
		}

		for _, group := range job.TaskGroups {
			if *group.Name == packet.InstanceType {
				if group.Count == nil {
					group.Count = &packet.Quantity // if 0 instances, increase set
				} else {
					*group.Count += packet.Quantity
				}
			}
		}

		_, _, err := client.Jobs().Register(job, nil)
		if err != nil {
			log.Printf("Error registering job: %v", err)
			reponse, _ := json.Marshal(instances.InstanceCreateResponse{
				Success: false,
				Message: "JOB_REGISTRATION_FAILED",
			})
			err := msg.Respond(reponse)
			if err != nil {
				log.Printf("Error sending acknowledgment: %v", err)
			}
			return
		}

		reponse, _ := json.Marshal(instances.InstanceCreateResponse{
			Success: true,
			Message: "SUCCESS",
		})
		errRespond := msg.Respond(reponse)
		if errRespond != nil {
			log.Printf("Error sending acknowledgment: %v", err)
		}
	})
	if err != nil {
		log.Fatalf("Error subscribing to subject %s: %v", subject, err)
	}
	log.Printf("Listening for instance creations on subject '%s'", subject)
}
