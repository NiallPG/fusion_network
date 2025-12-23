package command

import "math"

type KalmanTracker struct {
	// State: [x, y, vx, vy]
	X  float64
	Y  float64
	VX float64
	VY float64

	// Uncertainty
	P [4][4]float64

	// Process noise
	Q float64

	// Measurement noise
	R float64

	initialized bool
}

func NewKalmanTracker(processNoise, measurementNoise float64) *KalmanTracker {
	return &KalmanTracker{
		Q:           processNoise,
		R:           measurementNoise,
		initialized: false,
	}
}

func (k *KalmanTracker) Initialize(x, y float64) {
	k.X = x
	k.Y = y
	k.VX = 0
	k.VY = 0

	// Initial uncertainty
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			if i == j {
				k.P[i][j] = 1.0
			} else {
				k.P[i][j] = 0
			}
		}
	}
	k.initialized = true
}

func (k *KalmanTracker) Predict(dt float64) {
	if !k.initialized {
		return
	}

	// State prediction: x = x + vx*dt, y = y + vy*dt
	k.X += k.VX * dt
	k.Y += k.VY * dt

	// Covariance prediction (simplified)
	k.P[0][0] += k.Q
	k.P[1][1] += k.Q
	k.P[2][2] += k.Q
	k.P[3][3] += k.Q
}

func (k *KalmanTracker) Update(measuredX, measuredY float64) {
	if !k.initialized {
		k.Initialize(measuredX, measuredY)
		return
	}

	// Innovation (measurement residual)
	innovX := measuredX - k.X
	innovY := measuredY - k.Y

	// Kalman gain (simplified for position-only measurement)
	kx := k.P[0][0] / (k.P[0][0] + k.R)
	ky := k.P[1][1] / (k.P[1][1] + k.R)

	// State update
	k.X += kx * innovX
	k.Y += ky * innovY

	// Estimate velocity from position change
	k.VX = 0.8*k.VX + 0.2*innovX
	k.VY = 0.8*k.VY + 0.2*innovY

	// Covariance update
	k.P[0][0] *= (1 - kx)
	k.P[1][1] *= (1 - ky)
}

func (k *KalmanTracker) GetState() (x, y, vx, vy float64) {
	return k.X, k.Y, k.VX, k.VY
}

func (k *KalmanTracker) GetUncertainty() float64 {
	return math.Sqrt(k.P[0][0] + k.P[1][1])
}

