package main

import (
	"context"
	"log"
	"math/rand"
	"proto"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Client struct {
	tc proto.TrainerClient
}

func (c Client) doTrain() error {
	start := timestamppb.Now()
	settings := &proto.Settings{
		MaxPos: 10,
		MinPos: 1,
	}
	deviceid := uuid.NewString()
	stream, err := c.tc.Train(context.Background())

	if err != nil {
		log.Fatalln(err)
	}

	for i := 0; i < 100; i++ {

		// Simulate Work

		time.Sleep(time.Millisecond * time.Duration(rand.Intn(10)))
		t := proto.Training{
			DeviceID:   deviceid,
			Devicetype: proto.DeviceType_Bizeps,
			Start:      start,
			Finish:     timestamppb.Now(),
			Settings:   settings,
			User:       "timw",
			Iterations: []*proto.Iteration{
				{Force: rand.Int31n(1000)},
				{Force: rand.Int31n(1000)},
				{Force: rand.Int31n(1000)},
				{Force: rand.Int31n(1000)},
				{Force: rand.Int31n(1000)},
			},
		}

		err = stream.Send(&t)
		if err != nil {
			log.Fatal(err)
		}
	}

	_, err = stream.CloseAndRecv()
	if err != nil {
		log.Fatalln(err)
	}
	// log.Println("Client side Sum:", sum)
	return nil
}
