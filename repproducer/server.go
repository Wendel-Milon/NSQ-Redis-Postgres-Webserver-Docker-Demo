package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"proto"
	"sync"
	"time"
)

type Server struct {
	proto.UnimplementedTrainerServer

	mu       sync.Mutex
	max, min int32
}

func (s *Server) Train(t proto.Trainer_TrainServer) error {
	var total int32
	var sum int32

	var deviceid string
	var start time.Time

	var last time.Time

	for {
		training, err := t.Recv()
		if total == 0 {
			deviceid = training.DeviceID
			start = training.Start.AsTime()
		}

		for _, Iteration := range training.GetIterations() {
			sum = sum + Iteration.Force
		}

		if err == io.EOF {
			resp := &proto.Summary{Force: sum}

			s.mu.Lock()
			if sum > s.max {
				s.max = sum
			}
			if sum < s.min {
				s.min = sum
			}
			s.mu.Unlock()
			fmt.Printf("Device %s produced %d Force, in %s\n", deviceid, sum, last.Sub(start))
			return t.SendAndClose(resp)
		}

		if err != nil {
			return err
		}
		total++

		last = training.Finish.AsTime()

		// log.Printf("%+v\n", training)
	}
}

func (s *Server) FullRandom(t proto.Trainer_FullRandomServer) error {
	var clientSum int32
	var clientMessages int32

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				fr, err := t.Recv()
				// fmt.Printf("*")
				if err != nil {
					return
				}
				clientSum = clientSum + fr.A + fr.B + fr.C
				clientMessages++
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
			err := t.Send(fr)
			if err != nil {
				return nil
			}
		}
	}

	log.Println("Server says: clientSum =", clientSum, "with ", clientMessages, "messages.")
	return nil
}
