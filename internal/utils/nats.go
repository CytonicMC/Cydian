package utils

import (
	"fmt"
	"os"
)

func NatsUrl() string {
	username := os.Getenv("NATS_USERNAME")
	password := os.Getenv("NATS_PASSWORD")
	hostname := os.Getenv("NATS_HOSTNAME")
	port := os.Getenv("NATS_PORT")

	return fmt.Sprintf("nats://%s:%s@%s:%s", username, password, hostname, port)
}
