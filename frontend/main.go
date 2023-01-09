package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	nsq "github.com/nsqio/go-nsq"
)

func main() {

	if len(os.Args) != 2 {
		log.Fatal("Enter either 'Consume' or 'Produce'")
	}

	if os.Args[1] == "Consume" {
		go ConsumeMessage("topic", "chan_One")
		ConsumeMessage("topic", "chan_Two")

		return
	}

	ProduceMessage()

	// CallServer()

}

type myMessageHandler struct{}

// HandleMessage implements the Handler interface.
func (h *myMessageHandler) HandleMessage(m *nsq.Message) error {

	log.Printf("%s\n", m.Body)

	if len(m.Body) == 0 {
		// Returning nil will automatically send a FIN command to NSQ to mark the message as processed.
		// In this case, a message with an empty body is simply ignored/discarded.
		return nil
	}

	// do whatever actual message processing is desired

	// Returning a non-nil error will automatically send a REQ command to NSQ to re-queue the message.
	return nil
}

// ConsumeMessage does currently not work!!
// This is (guessed) because it polls the NSQLookup which works,
// The Lookup returns the dockerinternal IP Addres to which the client has no acess too.
// To work this either has all be in Docker or nothing at all....
func ConsumeMessage(topic, channel string) {

	// Instantiate a consumer that will subscribe to the provided channel.
	config := nsq.NewConfig()
	consumer, err := nsq.NewConsumer(topic, channel, config)
	if err != nil {
		log.Fatal(err)
	}

	// Set the Handler for messages received by this Consumer. Can be called multiple times.
	// See also AddConcurrentHandlers.
	consumer.AddHandler(&myMessageHandler{})

	// Use nsqlookupd to discover nsqd instances.
	// See also ConnectToNSQD, ConnectToNSQDs, ConnectToNSQLookupds.
	err = consumer.ConnectToNSQLookupd("127.0.0.1:4161")
	if err != nil {
		log.Fatal(err)
	}

	// wait for signal to exit
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// Gracefully stop the consumer.
	consumer.Stop()

	// os.Exit(1)

}

func ProduceMessage() {
	// Instantiate a producer.
	config := nsq.NewConfig()
	producer, err := nsq.NewProducer("127.0.0.1:4150", config)
	if err != nil {
		log.Fatal(err)
	}
	i := 0

	go func() {

		for {
			time.Sleep(time.Second * 2)

			message := fmt.Sprintf("Hello iteration %d at Datetime %v", i, time.Now())
			topicName := "topic"

			// Synchronously publish a single message to the specified topic.
			// Messages can also be sent asynchronously and/or in batches.
			err = producer.Publish(topicName, []byte(message))
			if err != nil {
				log.Fatal(err)
			}

			err = producer.Publish("Unterhose", []byte("Ich brauche davon etwas neues!"))
			if err != nil {
				log.Fatal(err)
			}

			fmt.Println("Produced!", i)
			i++
		}

	}()

	// wait for signal to exit
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// Gracefully stop the producer when appropriate (e.g. before shutting down the service)
	producer.Stop()
	os.Exit(1)
}

func CallServer() {

	link := os.Getenv("BACKEND_LINK")
	fmt.Println(link)

	resp, err := http.Get(fmt.Sprintf("http://%s:8080", link))

	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%s\n", body)
	fmt.Println(resp.Request.URL)
}
