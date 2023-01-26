package main

import (
	"log"
	"math/rand"
	"net"
	"proto"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {

	rand.Seed(time.Now().Unix())

	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := Client{
		tc: proto.NewTrainerClient(conn),
	}

	log.Println("Starting Server")
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalln(err)
	}
	server := grpc.NewServer()
	protoServer := Server{min: 10000000}

	go func() {
		time.Sleep(time.Second)
		for i := 0; i < 100; i++ {
			go client.doTrain()
		}
		// time.Sleep(time.Second)
		// log.Println(protoServer.max, protoServer.min)
	}()

	proto.RegisterTrainerServer(server, &protoServer)
	go log.Fatalln(server.Serve(listener))
}
