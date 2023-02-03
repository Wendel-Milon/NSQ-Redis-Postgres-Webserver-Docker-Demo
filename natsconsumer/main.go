package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/nats-io/nats.go"
)

var NATS_URL = os.Getenv("NATS_URL")

func main() {
	// Connect to a server
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	for i := 0; i < 5; i++ {
		MyQueueSubscribe(i, "foo", "multi")
	}

	for i := 5; i < 10; i++ {
		MyQueueSubscribe(i, "foo", "grp2")
	}

	ReqListener(1)
	ReqListener(2)

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		go http.ListenAndServe(":2112", nil)
	}()

	done := make(chan bool)
	<-done
}

func MyQueueSubscribe(id int, subj, queue string) {
	nc, err := nats.Connect(fmt.Sprintf("%s:4222", NATS_URL))
	if err != nil {
		log.Error().Err(err).Msg("")
		return
	}
	log.Info().Str("URL", NATS_URL).Msg("Successfully connected.")
	nc.QueueSubscribe(subj, queue, func(m *nats.Msg) {
		log.Info().Int("ID", id).Str("Subject", subj).Str("queue", queue).Str("Message", string(m.Data)).Msg("")
	})
}

func ReqListener(id int) {
	nc, err := nats.Connect(fmt.Sprintf("%s:4222", NATS_URL))
	if err != nil {
		log.Error().Err(err).Msg("")
		return
	}
	log.Info().Str("URL", NATS_URL).Msg("Successfully connected.")

	nc.Subscribe("*", func(m *nats.Msg) {
		nc.Publish(m.Reply, []byte(fmt.Sprint(id)))
		log.Info().Int("ID", id).Str("Subject", "*").Str("Message", string(m.Data)).Msg("")
	})
}
