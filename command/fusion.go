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
}

func NewFusionEngine(clusterRadius float64, minSensors int) *FusionEngine {
	return &FusionEngine{
		clusterRadius:  clusterRadius,
		minSensors:     minSensors,
		threats:        make(map[int]*FusedThreat),
		nextID:         1,
		expirationTime: 2 * time.Second,
	}
}

func (f *FusionEngine) ProcessReading(reading *sensorpb.SensorReading) *FusedThreat {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Try to match to existing threat
	for _, threat := range f.threats {
		dist := f.distance(reading.X, reading.Y, threat.X, threat.Y)
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
	var sumX, sumY, sumConf float64
	var sumLevel int

	for _, r := range threat.Readings {
		sumX += r.X * r.Confidence
		sumY += r.Y * r.Confidence
		sumConf += r.Confidence
		sumLevel += int(r.Level)
	}

	threat.X = sumX / sumConf
	threat.Y = sumY / sumConf
	threat.Confidence = sumConf / float64(len(threat.Readings))
	threat.Level = sumLevel / len(threat.Readings)
	threat.LastSeen = time.Now()
}

func (f *FusionEngine) distance(x1, y1, x2, y2 float64) float64 {
	dx := x2 - x1
	dy := y2 - y1
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
