package main

import (
	"context"
	"math/rand"
	"proto"
	"time"

	"github.com/rs/zerolog/log"

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
		log.Fatal().Err(err).Msg("")
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
			log.Fatal().Err(err).Msg("")
		}
	}

	_, err = stream.CloseAndRecv()
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}
	// log.Info().Msgf("Client side Sum:", sum)
	return nil
}

func (c Client) fullRandom() error {
	var serverSum int32
	var serverMessages int32

	stream, err := c.tc.FullRandom(context.Background())
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				fr, err := stream.Recv()
				if err != nil {
					return
				}
				serverSum = serverSum + fr.A + fr.B + fr.C
				serverMessages++
			}

		}
	}()
raus:
	for {
		select {
		case <-ctx.Done():
			break raus
		default:
			fr := &proto.Nums{
				A: rand.Int31n(100),
				B: rand.Int31n(100),
				C: rand.Int31n(100),
			}
			err := stream.Send(fr)
			if err != nil {
				return nil
			}
		}
	}

	defer log.Info().Msgf("Client says: serverSum = %d with  %s, messages.", serverSum, serverMessages)
	return nil
}
