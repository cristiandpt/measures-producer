package main

import (
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
	"errors"
	"context"
	"os"
	"sync"
	"time"
)

func main () {
	queueMeasures := "measures_queue"
	addr := "amqp://guest:guest@localhost:5672/"
	queue := NewQueue(queueMeasures, addr)


}
