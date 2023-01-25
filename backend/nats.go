package main

import (
	"net/http"
	"proto"

	pb "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (server *Server) NatsPost(w http.ResponseWriter, r *http.Request) {

	msg := proto.Person{
		Name:  "Backend",
		Id:    123,
		Email: "test.test@test.test",
		Phones: []*proto.Person_PhoneNumber{{
			Number: "asdfosadüfo",
			Type:   proto.Person_MOBILE,
		}, {
			Number: "0000111111",
			Type:   proto.Person_HOME,
		}},
		LastUpdated: timestamppb.Now(),
	}

	msgMarsh, err := pb.Marshal(&msg)
	if err != nil {
		server.SendErrorMessage(w, r, http.StatusBadRequest, err.Error())
	}

	// logrus.Info(len(msgMarsh), len(msgJSON))

	err = server.nats.Publish("foo", msgMarsh)
	if err != nil {
		server.SendErrorMessage(w, r, 404, err.Error())
	}
	// log.Println("Received POST!")
}
