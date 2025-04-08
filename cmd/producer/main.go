package main

import (
	"context"
	"fmt"
	actor "github.com/cristiandpt/measures-producer/internal/actor"
	"log"
	"os"
	"time"
)

func main() {
	queueMeasures := "measures_queue"
	addr := os.Getenv("RABBITMQ_ADDR") //"amqp://guest:guest@localhost:5672/"
	rabbitActor := actor.NewRabbitMQActor(queueMeasures, addr)
	defer rabbitActor.Close()

	// Testing message
	message := []byte("actor_message")
	currentTime := time.Now()
	messageTimestanded := []byte(fmt.Sprintf("[Message]: %s at -> %s", message, currentTime.Format("2006-01-02 15:04:05")))

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second*20))
	defer cancel()

loop:
	for {
		select {
		// Attempt to push a message every 2 seconds by sending a message to the actor
		case <-time.After(time.Second * 2):
			rabbitActor.Push(messageTimestanded)
			log.Println("Message sent to actor for pushing.")
		case <-ctx.Done():
			log.Println("Context done. Shutting down...")
			break loop
		}
	}

	log.Println("Main function finished.")
}
