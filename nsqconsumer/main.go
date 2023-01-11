package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nsqio/go-nsq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var NSQ_LOOKUP = os.Getenv("NSQ_LOOKUP")
var NSQ_CHAN = os.Getenv("NSQ_CHAN")
var NSQ_TOPIC = os.Getenv("NSQ_TOPIC")

func main() {
	log.Println("Starting consuming for Lookup:", NSQ_LOOKUP, "; Topic:", NSQ_TOPIC, "; Channel:", "NSQ_CHAN")
	ConsumeMessage()
}

type myMessageHandler struct{}

// HandleMessage implements the Handler interface.
func (h *myMessageHandler) HandleMessage(m *nsq.Message) error {

	if len(m.Body) == 0 {
		// Returning nil will automatically send a FIN command to NSQ to mark the message as processed.
		// In this case, a message with an empty body is simply ignored/discarded.
		return nil
	}

	// do whatever actual message processing is desired
	time.Sleep(time.Second * 10)
	log.Printf("%s\n", m.Body)

	// Returning a non-nil error will automatically send a REQ command to NSQ to re-queue the message.
	return nil
}

// ConsumeMessage does currently not work!!
// This is (guessed) because it polls the NSQLookup which works,
// The Lookup returns the dockerinternal IP Addres to which the client has no acess too.
// To work this either has all be in Docker or nothing at all....
func ConsumeMessage() {

	// Instantiate a consumer that will subscribe to the provided channel.
	config := nsq.NewConfig()
	consumer, err := nsq.NewConsumer(NSQ_TOPIC, NSQ_CHAN, config)
	if err != nil {
		log.Fatal(err)
	}

	// Set the Handler for messages received by this Consumer. Can be called multiple times.
	// See also AddConcurrentHandlers.
	consumer.AddHandler(&myMessageHandler{})

	// Use nsqlookupd to discover nsqd instances.
	// See also ConnectToNSQD, ConnectToNSQDs, ConnectToNSQLookupds.
	err = consumer.ConnectToNSQLookupd(fmt.Sprintf("%s:4161", NSQ_LOOKUP))
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(":2112", nil)
	}()

	// wait for signal to exit
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// Gracefully stop the consumer.
	consumer.Stop()

	// os.Exit(1)

}
