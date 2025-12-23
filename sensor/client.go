package sensor

import (
	"context"
	"log"

	sensorpb "distributed-sensor-fusion/shared/generated/sensorpb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type CommandClient struct {
	addr   string
	conn   *grpc.ClientConn
	stream sensorpb.SensorService_StreamReadingsClient
}

func NewCommandClient(addr string) *CommandClient {
	return &CommandClient{addr: addr}
}

func (c *CommandClient) Connect(ctx context.Context) error {
	conn, err := grpc.Dial(c.addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	c.conn = conn

	client := sensorpb.NewSensorServiceClient(conn)
	stream, err := client.StreamReadings(ctx)
	if err != nil {
		return err
	}
	c.stream = stream

	log.Printf("Connected to command server at %s", c.addr)
	return nil
}

func (c *CommandClient) Send(reading *sensorpb.SensorReading) error {
	return c.stream.Send(reading)
}

func (c *CommandClient) Close() error {
	if c.stream != nil {
		c.stream.CloseAndRecv()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
