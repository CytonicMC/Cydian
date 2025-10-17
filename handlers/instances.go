package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/CytonicMC/Cydian/env"
	"github.com/CytonicMC/Cydian/instances"
	"github.com/hashicorp/nomad/api"
	"github.com/nats-io/nats.go"
)

func RegisterInstances(nc *nats.Conn) {
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		log.Fatal(err)
	}
	createHandler(nc, client)
	deleteAllHandler(nc, client)
	deleteHandler(nc, client)
	updateHandler(nc, client)
}

func createHandler(nc *nats.Conn, client *api.Client) {
	const subject = "servers.create"
	_, err := nc.Subscribe(env.EnsurePrefixed(subject), func(msg *nats.Msg) {
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

		count := 0

		for _, group := range job.TaskGroups {
			if *group.Name == packet.InstanceType {
				if group.Count == nil {
					count = packet.Quantity
					group.Count = &packet.Quantity // if 0 instances, increase set
				} else {
					count = *group.Count + packet.Quantity
				}
			}
		}

		_, _, err := client.Jobs().Scale(*job.ID, packet.InstanceType, &count, "Adding instance(s)", false, nil, nil)

		if err != nil {
			reponse, _ := json.Marshal(instances.InstanceResponse{
				Success: false,
				Message: "JOB_SCALING_FAILED",
			})
			log.Printf("Error scaling job: %v", err)
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
	_, err := nc.Subscribe(env.EnsurePrefixed(subject), func(msg *nats.Msg) {
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
		zero := 0
		for _, group := range job.TaskGroups {
			if *group.Name == packet.InstanceType {
				if group.Count == nil {

					group.Count = &zero
				} else {
					*group.Count = 0
				}
			}
		}

		_, _, err := client.Jobs().Scale(*job.ID, packet.InstanceType, &zero, "Removing all instances", false, nil, nil)
		if err != nil {
			if err != nil {
				log.Printf("Error registering job: %v", err)
				reponse, _ := json.Marshal(instances.InstanceResponse{
					Success: false,
					Message: "SCALE_TO_ZERO_FAILED",
				})
				err := msg.Respond(reponse)
				if err != nil {
					log.Printf("Error sending acknowledgment: %v", err)
				}
				return
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
	log.Printf("Listening for bulk instance deletions on subject '%s'", subject)
}

func deleteHandler(nc *nats.Conn, client *api.Client) {
	const subject = "servers.delete"
	_, err := nc.Subscribe(env.EnsurePrefixed(subject), func(msg *nats.Msg) {
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
					zero := 0
					group.Count = &zero
				} else {
					*group.Count -= 1
				}
			}
		}

		_, _, err2 := client.Jobs().Register(job, nil)
		if err2 != nil {
			log.Printf("Error registering job: %v", err2)
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
			log.Printf("Error sending acknowledgment: %v", errRespond)
		}
	})
	if err != nil {
		log.Fatalf("Error subscribing to subject %s: %v", subject, err)
	}
	log.Printf("Listening for instance deletions on subject '%s'", subject)
}

func updateHandler(nc *nats.Conn, client *api.Client) {
	//todo: graceful server updates
	const subject = "servers.update"
	_, err := nc.Subscribe(env.EnsurePrefixed(subject), func(msg *nats.Msg) {
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

		job.Meta = map[string]string{
			"update_trigger": fmt.Sprintf("%d", time.Now().UnixNano()),
		}

		_, _, err := client.Jobs().Register(job, nil)

		if err != nil {
			reponse, _ := json.Marshal(instances.InstanceResponse{
				Success: false,
				Message: "JOB_REGISTRATION_FAILED",
			})
			log.Printf("Error scaling job: %v", err)
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
	log.Printf("Listening for instance updates on subject '%s'", subject)
}
