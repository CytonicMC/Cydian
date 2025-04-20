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
	deleteAllHandler(nc, client)
	deleteHandler(nc, client)
}

func createHandler(nc *nats.Conn, client *api.Client) {
	const subject = "servers.create"
	_, err := nc.Subscribe(subject, func(msg *nats.Msg) {
		var packet instances.InstanceCreateRequest
		if err := json.Unmarshal(msg.Data, &packet); err != nil {
			log.Printf("Invalid message format: %s", msg.Data)
			response, _ := json.Marshal(instances.InstanceResponse{
				Success: false,
				Message: "INVALID_MESSAGE_FORMAT",
			})
			err := msg.Respond(response)
			if err != nil {
				log.Printf("Error sending acknowledgment: %v", err)
			}
			return
		}

		job, _, errJobs := client.Jobs().Info(packet.InstanceType, nil)
		if errJobs != nil {
			log.Printf("Error getting job info: %v", errJobs)
			reponse, _ := json.Marshal(instances.InstanceResponse{
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
			reponse, _ := json.Marshal(instances.InstanceResponse{
				Success: false,
				Message: "JOB_REGISTRATION_FAILED",
			})
			err := msg.Respond(reponse)
			if err != nil {
				log.Printf("Error sending acknowledgment: %v", err)
			}
			return
		}

		reponse, _ := json.Marshal(instances.InstanceResponse{
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

func deleteAllHandler(nc *nats.Conn, client *api.Client) {
	const subject = "servers.delete.all"
	_, err := nc.Subscribe(subject, func(msg *nats.Msg) {
		var packet instances.InstanceDeleteAllRequest
		if err := json.Unmarshal(msg.Data, &packet); err != nil {
			log.Printf("Invalid message format: %s", msg.Data)
			response, _ := json.Marshal(instances.InstanceResponse{
				Success: false,
				Message: "INVALID_MESSAGE_FORMAT",
			})
			err := msg.Respond(response)
			if err != nil {
				log.Printf("Error sending acknowledgment: %v", err)
			}
			return
		}

		allocs, _, errallocs := client.Jobs().Allocations(packet.InstanceType, true, nil)
		if errallocs != nil {
			log.Printf("Error getting job info: %v", errallocs)
			reponse, _ := json.Marshal(instances.InstanceResponse{
				Success: false,
				Message: "JOB_NOT_FOUND",
			})
			err := msg.Respond(reponse)
			if err != nil {
				log.Printf("Error sending acknowledgment: %v", err)
			}
			return
		}

		for _, allocStub := range allocs {
			alloc, _, err := client.Allocations().Info(allocStub.ID, nil)
			if err != nil {
				log.Printf("Failed to fetch allocation %s: %v", allocStub.ID, err)
				continue
			}

			_, err = client.Allocations().Stop(alloc, nil)
			if err != nil {
				log.Printf("Failed to stop allocation %s: %v", alloc.ID, err)
				response, _ := json.Marshal(instances.InstanceResponse{
					Success: false,
					Message: "FAILED_TO_STOP_ALLOCATION_" + alloc.ID,
				})
				err := msg.Respond(response)
				if err != nil {
					log.Printf("Error sending acknowledgment: %v", err)
				}
				return
			} else {
				log.Printf("Stopped allocation %s", alloc.ID)
			}
		}

		reponse, _ := json.Marshal(instances.InstanceResponse{
			Success: true,
			Message: "SUCCESS",
		})
		errRespond := msg.Respond(reponse)
		if errRespond != nil {
			log.Printf("Error sending acknowledgment: %v", errRespond)
		}
	})
	if err != nil {
		log.Fatalf("Error subscribing to subject %s: %v", subject, err)
	}
	log.Printf("Listening for instance creations on subject '%s'", subject)
}

func deleteHandler(nc *nats.Conn, client *api.Client) {
	const subject = "servers.delete"
	_, err := nc.Subscribe(subject, func(msg *nats.Msg) {
		var packet instances.InstanceDeleteRequest
		if err := json.Unmarshal(msg.Data, &packet); err != nil {
			log.Printf("Invalid message format: %s", msg.Data)
			response, _ := json.Marshal(instances.InstanceResponse{
				Success: false,
				Message: "INVALID_MESSAGE_FORMAT",
			})
			err := msg.Respond(response)
			if err != nil {
				log.Printf("Error sending acknowledgment: %v", err)
			}
			return
		}

		alloc, _, err := client.Allocations().Info(packet.AllocId, nil)
		if err != nil {
			log.Printf("Failed to fetch allocation %s: %v", packet.AllocId, err)
			response, _ := json.Marshal(instances.InstanceResponse{
				Success: false,
				Message: "ALLOCATION_NOT_FOUND",
			})
			err := msg.Respond(response)
			if err != nil {
				log.Printf("Error sending acknowledgment: %v", err)
			}
			return
		}

		_, err = client.Allocations().Stop(alloc, nil)
		if err != nil {
			log.Printf("Failed to stop allocation %s: %v", alloc.ID, err)
			response, _ := json.Marshal(instances.InstanceResponse{
				Success: false,
				Message: "FAILED_TO_STOP_ALLOCATION",
			})
			err := msg.Respond(response)
			if err != nil {
				log.Printf("Error sending acknowledgment: %v", err)
			}
			return
		} else {
			log.Printf("Stopped allocation %s", alloc.ID)
		}

		reponse, _ := json.Marshal(instances.InstanceResponse{
			Success: true,
			Message: "SUCCESS",
		})
		errRespond := msg.Respond(reponse)
		if errRespond != nil {
			log.Printf("Error sending acknowledgment: %v", errRespond)
		}
	})
	if err != nil {
		log.Fatalf("Error subscribing to subject %s: %v", subject, err)
	}
	log.Printf("Listening for instance creations on subject '%s'", subject)
}
