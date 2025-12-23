package world

import (
	"math/rand"
)

type Threat struct {
	X     float64
	Y     float64
	VX    float64
	VY    float64
	ID    int
	Level int
}

type World struct {
	Threats []Threat
	Tick    int
	Width   float64
	Height  float64
}

func NewWorld(numThreats int, width, height float64) *World {
	return &World{
		Threats: createThreats(numThreats, width, height),
		Tick:    0,
		Width:   width,
		Height:  height,
	}
}

func createThreats(numThreats int, width, height float64) []Threat {
	threats := make([]Threat, numThreats)
	for i := 0; i < numThreats; i++ {
		threats[i] = Threat{
			X:     rand.Float64() * width,
			Y:     rand.Float64() * height,
			VX:    (rand.Float64() * 4) - 2,
			VY:    (rand.Float64() * 4) - 2,
			ID:    i,
			Level: rand.Intn(10) + 1,
		}
	}
	return threats
}

func (w *World) Step() {
	for i := range w.Threats {
		UpdatePosition(&w.Threats[i], w.Width, w.Height)
	}
	w.Tick++
}
