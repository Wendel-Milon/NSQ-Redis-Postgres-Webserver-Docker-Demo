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

	go func() {
		nc, err := nats.Connect(fmt.Sprintf("%s:4222", NATS_URL))
		if err != nil {
			log.Fatal().Err(err).Msg("")
		}

		nc.QueueSubscribe("foo", "multi", func(m *nats.Msg) {
			log.Info().Int("ID", 1).Str("Message", string(m.Data)).Msg("")
		})
		nc.QueueSubscribe("foo", "multi", func(m *nats.Msg) {
			log.Info().Int("ID", 2).Str("Message", string(m.Data)).Msg("")
		})
	}()

	go func() {
		nc, err := nats.Connect(fmt.Sprintf("%s:4222", NATS_URL))
		if err != nil {
			log.Fatal().Err(err).Msg("")
		}

		nc.QueueSubscribe("foo", "multi", func(m *nats.Msg) {
			log.Info().Int("ID", 3).Str("Message", string(m.Data)).Msg("")
		})
		nc.QueueSubscribe("foo", "multi", func(m *nats.Msg) {
			log.Info().Int("ID", 4).Str("Message", string(m.Data)).Msg("")
		})
	}()

	go func() {
		nc, err := nats.Connect(fmt.Sprintf("%s:4222", NATS_URL))
		if err != nil {
			log.Fatal().Err(err).Msg("")
		}

		nc.QueueSubscribe("foo", "grp2", func(m *nats.Msg) {
			log.Warn().Int("ID", 5).Str("Message", string(m.Data)).Msg("")
		})
		nc.QueueSubscribe("foo", "grp2", func(m *nats.Msg) {
			log.Warn().Int("ID", 6).Str("Message", string(m.Data)).Msg("")
		})
	}()

	// nc, err := nats.Connect(nats.DefaultURL)
	// if err != nil {
	// 	log.Fatal().Err(err).Msg("")
	// }

	// time.Sleep(time.Second)
	// for i := 0; i < 100; i++ {
	// 	// Simple Publisher
	// 	err = nc.Publish("foo.TEST", []byte(fmt.Sprintf("%d", i)))
	// 	if err != nil {
	// 		log.Fatal().Err(err).Msg("")
	// 	}
	// }

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		go http.ListenAndServe(":2112", nil)
	}()

	done := make(chan bool)
	<-done
}
