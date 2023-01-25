package main

import (
	"context"
	"net/http"
	"proto"

	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/encoding/protojson"
)

func (s *Server) CallGRPCPost(w http.ResponseWriter, r *http.Request) {
	client := proto.NewGreeterClient(s.grpc)

	reply, err := client.SayHello(context.Background(), &proto.HelloRequest{
		Name: "SHAALALALL",
	})
	if err != nil {
		s.SendErrorMessage(w, r, http.StatusBadRequest, err.Error())
		logrus.Error(err)
	}

	bytes, err := protojson.Marshal(reply)
	if err != nil {
		s.SendErrorMessage(w, r, http.StatusBadRequest, err.Error())
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_, err = w.Write(bytes)
	if err != nil {
		s.SendError(w, r)
		return
	}
}
