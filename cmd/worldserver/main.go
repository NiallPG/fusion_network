package main

import (
	"log"
	"time"

	"distributed-sensor-fusion/world"
)

const tickRate = 500 * time.Millisecond

func main() {
	server := world.NewWorldServer(3, 100.0, 100.0)

	go server.RunSimulation(tickRate)
	go server.StartControlServer(":8081")

	if err := server.Start(":50051"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

