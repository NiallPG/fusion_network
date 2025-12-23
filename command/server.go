package command

import (
	"io"
	"log"
	"net"
	"sync"

	sensorpb "distributed-sensor-fusion/shared/generated/sensorpb"

	"google.golang.org/grpc"
)

type CommandServer struct {
	sensorpb.UnimplementedSensorServiceServer
	fusion     *FusionEngine
	trackers   map[int]*KalmanTracker
	trackersMu sync.RWMutex
	broadcast  chan *FusedThreat
}

func NewCommandServer(clusterRadius float64, minSensors int) *CommandServer {
	return &CommandServer{
		fusion:    NewFusionEngine(clusterRadius, minSensors),
		trackers:  make(map[int]*KalmanTracker),
		broadcast: make(chan *FusedThreat, 100),
	}
}

func (s *CommandServer) StreamReadings(stream sensorpb.SensorService_StreamReadingsServer) error {
	for {
		reading, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&sensorpb.Ack{Received: true})
		}
		if err != nil {
			return err
		}

		log.Printf("Received from %s: threat=%d pos=(%.1f, %.1f)",
			reading.SensorId, reading.ThreatId, reading.X, reading.Y)

		// Process through fusion engine
		if confirmed := s.fusion.ProcessReading(reading); confirmed != nil {
			// Apply Kalman filter
			s.applyTracking(confirmed)

			// Broadcast to WebSocket clients
			select {
			case s.broadcast <- confirmed:
			default:
			}
		}
	}
}

func (s *CommandServer) applyTracking(threat *FusedThreat) {
	s.trackersMu.Lock()
	defer s.trackersMu.Unlock()

	tracker, exists := s.trackers[threat.ID]
	if !exists {
		tracker = NewKalmanTracker(0.1, 1.0)
		s.trackers[threat.ID] = tracker
	}

	tracker.Predict(0.1) // Assume 100ms between updates
	tracker.Update(threat.X, threat.Y)

	// Update threat with smoothed position
	threat.X, threat.Y, _, _ = tracker.GetState()
}

func (s *CommandServer) Broadcast() <-chan *FusedThreat {
	return s.broadcast
}

func (s *CommandServer) Fusion() *FusionEngine {
	return s.fusion
}

func (s *CommandServer) Start(port string) error {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()
	sensorpb.RegisterSensorServiceServer(grpcServer, s)

	log.Printf("Command server listening on %s", port)
	return grpcServer.Serve(lis)
}
