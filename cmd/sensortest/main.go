package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"distributed-sensor-fusion/sensor"
)

func main() {
	sensorID := "sensor-1"
	if len(os.Args) > 1 {
		sensorID = os.Args[1]
	}

	noise := sensor.DefaultNoiseConfig()
	s := sensor.NewSensor(sensorID, "localhost:50051", noise)
	client := sensor.NewCommandClient("localhost:50052")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Connect to command server
	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect to command: %v", err)
	}
	defer client.Close()

	// Forward readings to command server
	go func() {
		for reading := range s.Readings() {
			log.Printf("[%s] Sending: threat=%d pos=(%.1f, %.1f)",
				reading.SensorId, reading.ThreatId, reading.X, reading.Y)
			if err := client.Send(reading); err != nil {
				log.Printf("Send error: %v", err)
			}
		}
	}()

	// Start observing world
	if err := s.Start(ctx); err != nil {
		log.Fatalf("Sensor error: %v", err)
	}
}
