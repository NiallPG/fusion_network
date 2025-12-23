package main

import (
	"log"
	"time"

	"distributed-sensor-fusion/command"
)

func main() {
	// Cluster readings within 10 units, require 2+ sensors to confirm
	server := command.NewCommandServer(10.0, 2)
	wsHub := command.NewWebSocketHub()

	// Forward confirmed threats to WebSocket
	go func() {
		for threat := range server.Broadcast() {
			log.Printf("CONFIRMED: Threat %d at (%.1f, %.1f) level=%d sensors=%d",
				threat.ID, threat.X, threat.Y, threat.Level, threat.SensorCount)
			wsHub.BroadcastThreat(threat)
		}
	}()

	// Periodic cleanup of stale threats
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		for range ticker.C {
			server.Fusion().Cleanup()
		}
	}()

	// Start WebSocket server
	go wsHub.Start(":8080")

	// Start gRPC server (blocks)
	if err := server.Start(":50052"); err != nil {
		log.Fatalf("Failed to start command server: %v", err)
	}
}

