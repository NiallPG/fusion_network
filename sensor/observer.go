package sensor

import (
	"context"
	"io"
	"log"
	"math/rand"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	sensorpb "distributed-sensor-fusion/shared/generated/sensorpb"
	worldpb "distributed-sensor-fusion/shared/generated/worldpb"
)

type Sensor struct {
	ID          string
	NoiseConfig NoiseConfig
	WorldAddr   string
	readings    chan *sensorpb.SensorReading
}

func NewSensor(id string, worldAddr string, noise NoiseConfig) *Sensor {
	return &Sensor{
		ID:          id,
		NoiseConfig: noise,
		WorldAddr:   worldAddr,
		readings:    make(chan *sensorpb.SensorReading, 100),
	}
}

func (s *Sensor) Readings() <-chan *sensorpb.SensorReading {
	return s.readings
}

func (s *Sensor) Start(ctx context.Context) error {
	conn, err := grpc.Dial(s.WorldAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()

	client := worldpb.NewWorldServiceClient(conn)
	stream, err := client.Subscribe(ctx, &worldpb.SubscribeRequest{})
	if err != nil {
		return err
	}

	log.Printf("[%s] Connected to world server", s.ID)

	for {
		state, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		s.processWorldState(state)
	}

	return nil
}

func (s *Sensor) processWorldState(state *worldpb.WorldState) {
	now := time.Now().UnixNano()

	// Process real threats (with possible misses)
	for _, threat := range state.Threats {
		if s.NoiseConfig.ShouldMiss() {
			continue
		}

		noisyX, noisyY := s.NoiseConfig.AddPositionNoise(threat.X, threat.Y)
		noisyLevel := s.NoiseConfig.AddLevelNoise(int(threat.Level))

		reading := &sensorpb.SensorReading{
			SensorId:   s.ID,
			ThreatId:   threat.Id,
			X:          noisyX,
			Y:          noisyY,
			Level:      int32(noisyLevel),
			Timestamp:  now,
			Confidence: 0.7 + rand.Float64()*0.3, // 0.7 to 1.0
		}

		select {
		case s.readings <- reading:
		default:
			log.Printf("[%s] Reading channel full, dropping", s.ID)
		}
	}

	// Generate false positives
	if s.NoiseConfig.ShouldFalsePositive() {
		reading := &sensorpb.SensorReading{
			SensorId:   s.ID,
			ThreatId:   -1, // Fake threat
			X:          rand.Float64() * 100,
			Y:          rand.Float64() * 100,
			Level:      int32(rand.Intn(5) + 1),
			Timestamp:  now,
			Confidence: 0.3 + rand.Float64()*0.4, // 0.3 to 0.7
		}

		select {
		case s.readings <- reading:
		default:
		}
	}
}
