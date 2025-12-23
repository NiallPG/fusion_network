package command

// clustering engine
import (
	"math"
	"sync"
	"time"

	sensorpb "distributed-sensor-fusion/shared/generated/sensorpb"
)

type FusedThreat struct {
	ID          int
	X           float64
	Y           float64
	Level       int
	Confidence  float64
	SensorCount int
	LastSeen    time.Time
	Readings    []*sensorpb.SensorReading
}

type FusionEngine struct {
	mu             sync.RWMutex
	clusterRadius  float64
	minSensors     int
	threats        map[int]*FusedThreat
	nextID         int
	expirationTime time.Duration
	worldWidth     float64
	worldHeight    float64
}

func NewFusionEngine(clusterRadius float64, minSensors int) *FusionEngine {
	return &FusionEngine{
		clusterRadius:  clusterRadius,
		minSensors:     minSensors,
		threats:        make(map[int]*FusedThreat),
		nextID:         1,
		expirationTime: 2 * time.Second,
		worldWidth:     100.0,
		worldHeight:    100.0,
	}
}

func (f *FusionEngine) ProcessReading(reading *sensorpb.SensorReading) *FusedThreat {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Try to match to existing threat
	for _, threat := range f.threats {
		dist := f.wrappedDistance(reading.X, reading.Y, threat.X, threat.Y)
		if dist <= f.clusterRadius {
			f.updateThreat(threat, reading)
			if threat.SensorCount >= f.minSensors {
				return threat
			}
			return nil
		}
	}

	// No match - create pending threat
	threat := &FusedThreat{
		ID:          f.nextID,
		X:           reading.X,
		Y:           reading.Y,
		Level:       int(reading.Level),
		Confidence:  reading.Confidence,
		SensorCount: 1,
		LastSeen:    time.Now(),
		Readings:    []*sensorpb.SensorReading{reading},
	}
	f.threats[f.nextID] = threat
	f.nextID++

	return nil
}

func (f *FusionEngine) updateThreat(threat *FusedThreat, reading *sensorpb.SensorReading) {
	// Check if this sensor already contributed
	for _, r := range threat.Readings {
		if r.SensorId == reading.SensorId {
			// Update existing reading from this sensor
			r.X = reading.X
			r.Y = reading.Y
			r.Level = reading.Level
			r.Confidence = reading.Confidence
			r.Timestamp = reading.Timestamp
			f.recalculate(threat)
			return
		}
	}

	// New sensor contributing
	threat.Readings = append(threat.Readings, reading)
	threat.SensorCount = len(threat.Readings)
	f.recalculate(threat)
}

func (f *FusionEngine) recalculate(threat *FusedThreat) {
	var sumLevel int
	var sumConf float64

	// Use circular mean for positions to handle wrap-around
	var sinX, cosX, sinY, cosY float64
	for _, r := range threat.Readings {
		// Convert to angles (0-100 maps to 0-2π)
		angleX := (r.X / f.worldWidth) * 2 * math.Pi
		angleY := (r.Y / f.worldHeight) * 2 * math.Pi

		// Weight by confidence
		sinX += math.Sin(angleX) * r.Confidence
		cosX += math.Cos(angleX) * r.Confidence
		sinY += math.Sin(angleY) * r.Confidence
		cosY += math.Cos(angleY) * r.Confidence

		sumConf += r.Confidence
		sumLevel += int(r.Level)
	}

	// Convert back from angles to positions
	avgAngleX := math.Atan2(sinX, cosX)
	avgAngleY := math.Atan2(sinY, cosY)

	// Normalize to 0-100 range
	threat.X = (avgAngleX / (2 * math.Pi)) * f.worldWidth
	if threat.X < 0 {
		threat.X += f.worldWidth
	}
	threat.Y = (avgAngleY / (2 * math.Pi)) * f.worldHeight
	if threat.Y < 0 {
		threat.Y += f.worldHeight
	}

	threat.Confidence = sumConf / float64(len(threat.Readings))
	threat.Level = sumLevel / len(threat.Readings)
	threat.LastSeen = time.Now()
}

// wrappedDistance calculates distance accounting for toroidal wrap-around
func (f *FusionEngine) wrappedDistance(x1, y1, x2, y2 float64) float64 {
	dx := math.Abs(x2 - x1)
	dy := math.Abs(y2 - y1)

	// Check if wrapping around is shorter
	if dx > f.worldWidth/2 {
		dx = f.worldWidth - dx
	}
	if dy > f.worldHeight/2 {
		dy = f.worldHeight - dy
	}

	return math.Sqrt(dx*dx + dy*dy)
}

func (f *FusionEngine) GetConfirmedThreats() []*FusedThreat {
	f.mu.RLock()
	defer f.mu.RUnlock()

	confirmed := make([]*FusedThreat, 0)
	for _, threat := range f.threats {
		if threat.SensorCount >= f.minSensors {
			confirmed = append(confirmed, threat)
		}
	}
	return confirmed
}

func (f *FusionEngine) Cleanup() {
	f.mu.Lock()
	defer f.mu.Unlock()

	now := time.Now()
	for id, threat := range f.threats {
		if now.Sub(threat.LastSeen) > f.expirationTime {
			delete(f.threats, id)
		}
	}
}