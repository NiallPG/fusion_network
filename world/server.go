package world

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	pb "distributed-sensor-fusion/shared/generated/worldpb"

	"google.golang.org/grpc"
)

type WorldServer struct {
	pb.UnimplementedWorldServiceServer
	world       *World
	mu          sync.RWMutex
	subscribers []chan *pb.WorldState
	subMu       sync.Mutex

	paused     bool
	pausedMu   sync.RWMutex
	numThreats int
	width      float64
	height     float64
	tickRate   time.Duration
}

func NewWorldServer(numThreats int, width, height float64) *WorldServer {
	return &WorldServer{
		world:       NewWorld(numThreats, width, height),
		subscribers: make([]chan *pb.WorldState, 0),
		paused:      false,
		numThreats:  numThreats,
		width:       width,
		height:      height,
	}
}

func (s *WorldServer) Subscribe(req *pb.SubscribeRequest, stream pb.WorldService_SubscribeServer) error {
	ch := make(chan *pb.WorldState, 10)

	s.subMu.Lock()
	s.subscribers = append(s.subscribers, ch)
	s.subMu.Unlock()

	defer func() {
		s.subMu.Lock()
		for i, sub := range s.subscribers {
			if sub == ch {
				s.subscribers = append(s.subscribers[:i], s.subscribers[i+1:]...)
				break
			}
		}
		s.subMu.Unlock()
		close(ch)
	}()

	for state := range ch {
		if err := stream.Send(state); err != nil {
			return err
		}
	}
	return nil
}

func (s *WorldServer) broadcast() {
	s.mu.RLock()
	state := s.buildState()
	s.mu.RUnlock()

	s.subMu.Lock()
	for _, ch := range s.subscribers {
		select {
		case ch <- state:
		default:
		}
	}
	s.subMu.Unlock()
}

func (s *WorldServer) buildState() *pb.WorldState {
	threats := make([]*pb.Threat, len(s.world.Threats))
	for i, t := range s.world.Threats {
		threats[i] = &pb.Threat{
			Id:    int32(t.ID),
			X:     t.X,
			Y:     t.Y,
			Vx:    t.VX,
			Vy:    t.VY,
			Level: int32(t.Level),
		}
	}
	return &pb.WorldState{
		Threats: threats,
		Tick:    int32(s.world.Tick),
	}
}

func (s *WorldServer) RunSimulation(tickRate time.Duration) {
	s.pausedMu.Lock()
	s.tickRate = tickRate
	s.pausedMu.Unlock()

	ticker := time.NewTicker(tickRate)
	defer ticker.Stop()

	for range ticker.C {
		s.pausedMu.RLock()
		paused := s.paused
		s.pausedMu.RUnlock()

		if !paused {
			s.mu.Lock()
			s.world.Step()
			s.mu.Unlock()
		}

		s.broadcast()
	}
}

func (s *WorldServer) Pause() {
	s.pausedMu.Lock()
	defer s.pausedMu.Unlock()
	s.paused = true
	log.Println("Simulation paused")
}

func (s *WorldServer) Resume() {
	s.pausedMu.Lock()
	defer s.pausedMu.Unlock()
	s.paused = false
	log.Println("Simulation resumed")
}

func (s *WorldServer) Restart() {
	s.pausedMu.Lock()
	s.paused = false
	s.pausedMu.Unlock()

	s.mu.Lock()
	s.world = NewWorld(s.numThreats, s.width, s.height)
	s.mu.Unlock()

	log.Println("Simulation restarted")
}

func (s *WorldServer) IsPaused() bool {
	s.pausedMu.RLock()
	defer s.pausedMu.RUnlock()
	return s.paused
}

func (s *WorldServer) Start(port string) error {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()
	pb.RegisterWorldServiceServer(grpcServer, s)

	log.Printf("World server listening on %s", port)
	return grpcServer.Serve(lis)
}

func (s *WorldServer) StartControlServer(port string) {
	mux := http.NewServeMux()

	withCORS := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			h(w, r)
		}
	}

	mux.HandleFunc("/pause", withCORS(func(w http.ResponseWriter, r *http.Request) {
		s.Pause()
		s.writeStatus(w)
	}))

	mux.HandleFunc("/resume", withCORS(func(w http.ResponseWriter, r *http.Request) {
		s.Resume()
		s.writeStatus(w)
	}))

	mux.HandleFunc("/restart", withCORS(func(w http.ResponseWriter, r *http.Request) {
		s.Restart()
		s.writeStatus(w)
	}))

	mux.HandleFunc("/status", withCORS(func(w http.ResponseWriter, r *http.Request) {
		s.writeStatus(w)
	}))

	log.Printf("Control server listening on %s", port)
	if err := http.ListenAndServe(port, mux); err != nil {
		log.Fatalf("Control server error: %v", err)
	}
}

func (s *WorldServer) writeStatus(w http.ResponseWriter) {
	s.pausedMu.RLock()
	paused := s.paused
	s.pausedMu.RUnlock()

	s.mu.RLock()
	tick := s.world.Tick
	threatCount := len(s.world.Threats)
	s.mu.RUnlock()

	state := "running"
	if paused {
		state = "paused"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"state":   state,
		"tick":    tick,
		"threats": threatCount,
	})
}
