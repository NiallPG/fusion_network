package sensor

import (
	"math"
	"math/rand"
)

type NoiseConfig struct {
	PositionStdDev    float64 // Gaussian noise for X/Y
	FalsePositiveRate float64 // Probability of hallucinating a threat
	MissRate          float64 // Probability of missing a real threat
	LevelVariance     int     // How much threat level can vary
}

func DefaultNoiseConfig() NoiseConfig {
	return NoiseConfig{
		PositionStdDev:    3.0,
		FalsePositiveRate: 0.05,
		MissRate:          0.1,
		LevelVariance:     2,
	}
}

// Gaussian returns a normally distributed random number
func Gaussian(mean, stdDev float64) float64 {
	// Box-Muller transform
	u1 := rand.Float64()
	u2 := rand.Float64()
	z := math.Sqrt(-2*math.Log(u1)) * math.Cos(2*math.Pi*u2)
	return mean + z*stdDev
}

func (nc NoiseConfig) ShouldMiss() bool {
	return rand.Float64() < nc.MissRate
}

func (nc NoiseConfig) ShouldFalsePositive() bool {
	return rand.Float64() < nc.FalsePositiveRate
}

func (nc NoiseConfig) AddPositionNoise(x, y float64) (float64, float64) {
	return Gaussian(x, nc.PositionStdDev), Gaussian(y, nc.PositionStdDev)
}

func (nc NoiseConfig) AddLevelNoise(level int) int {
	noisy := level + rand.Intn(nc.LevelVariance*2+1) - nc.LevelVariance
	if noisy < 1 {
		return 1
	}
	if noisy > 10 {
		return 10
	}
	return noisy
}
