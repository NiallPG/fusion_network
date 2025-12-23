package sensor

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	sensorpb "distributed-sensor-fusion/shared/generated/sensorpb"

	"google.golang.org/grpc"
)

//

func main() {
	// ---- parse flags ----
	sensorID := flag.String("id", "", "Sensor ID")
	sensorType := flag.String("type", "", "Sensor type")
	serverAddr := flag.String("server", "localhost:50051", "Command server address")
	flag.Parse()

	// ---- connect to server ----
	conn, err := grpc.Dial(*serverAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	// ---- create SensorService client ----
	client := sensorpb.NewSensorServiceClient(conn)

	// ---- handle graceful shutdown ----
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// ---- start streaming loop ----
	_ = client
	_ = sensorID
	_ = sensorType
	_ = ctx
}
