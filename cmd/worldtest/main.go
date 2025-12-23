package main

import (
	"fmt"
	"time"

	"distributed-sensor-fusion/world"
)

func main() {
	w := world.NewWorld(3, 100.0, 100.0)

	for i := 0; i < 50; i++ {
		w.Step()
		fmt.Printf("Tick %d:\n", i)
		for _, t := range w.Threats {
			fmt.Printf("  ID=%d pos=(%.1f, %.1f) vel=(%.1f, %.1f)\n",
				t.ID, t.X, t.Y, t.VX, t.VY)
		}
		fmt.Println()
		time.Sleep(200 * time.Millisecond)
	}
}

