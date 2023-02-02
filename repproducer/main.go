package main

import (
	"math/rand"
	"net"
	"os"
	"proto"
	"time"

	"github.com/rs/zerolog/log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {

	rand.Seed(time.Now().Unix())

	// go ClientDoTrain()
	go ClientFullRandom()

	log.Info().Msgf("Starting Server")
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}
	server := grpc.NewServer()
	protoServer := Server{min: 10000000}

	proto.RegisterTrainerServer(server, &protoServer)
	go log.Fatal().Err(server.Serve(listener)).Msg("")
}

func ClientDoTrain() {
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal().Err(err).Msgf("")
	}
	defer conn.Close()

	client := Client{
		tc: proto.NewTrainerClient(conn),
	}

	time.Sleep(time.Second)

	for i := 0; i < 100; i++ {
		go client.doTrain()
	}
	time.Sleep(time.Second)
}

func ClientFullRandom() {
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal().Err(err).Msgf("")
	}
	defer conn.Close()

	client := Client{
		tc: proto.NewTrainerClient(conn),
	}

	time.Sleep(time.Second)

	err = client.fullRandom()
	if err != nil {
		log.Fatal().Err(err).Msgf("")
	}

	time.Sleep(time.Second)
	os.Exit(1)
}
