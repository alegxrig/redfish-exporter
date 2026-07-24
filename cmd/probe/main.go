package main

import (
	"fmt"
	"log"
	"os"

	"github.com/stmcginnis/gofish"
)

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	config := gofish.ClientConfig{
		Endpoint:  getenv("REDFISH_ENDPOINT", "http://127.0.0.1:8000"),
		Username:  getenv("REDFISH_USER", "root"),
		Password:  getenv("REDFISH_PASSWORD", "password"),
		Insecure:  true,
		BasicAuth: true, // мокап не умеет в session-based auth; на реальном BMC (iDRAC/iLO) это false
	}

	c, err := gofish.Connect(config)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer c.Logout()

	service := c.Service

	systems, err := service.Systems()
	if err != nil {
		log.Fatalf("systems: %v", err)
	}
	for _, sys := range systems {
		fmt.Printf("system=%q power=%s health=%s\n", sys.Name, sys.PowerState, sys.Status.Health)
	}

	chassisList, err := service.Chassis()
	if err != nil {
		log.Fatalf("chassis: %v", err)
	}
	for _, ch := range chassisList {
		thermal, err := ch.Thermal()
		if err != nil {
			log.Printf("chassis %q has no Thermal resource: %v", ch.Name, err)
			continue
		}
		for _, t := range thermal.Temperatures {
			if t.ReadingCelsius == nil {
				continue
			}
			fmt.Printf("chassis=%q sensor=%q celsius=%.1f\n", ch.Name, t.Name, *t.ReadingCelsius)
		}
	}
}
