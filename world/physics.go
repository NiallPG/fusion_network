package world

import "math"

func UpdatePosition(t *Threat, width, height float64) {
    t.X += t.VX
    t.Y += t.VY

    // wwrap around horizontally
    t.X = math.Mod(t.X, width)
    if t.X < 0 {
        t.X += width
    }

    // wrap around vertically
    t.Y = math.Mod(t.Y, height)
    if t.Y < 0 {
        t.Y += height
    }
}