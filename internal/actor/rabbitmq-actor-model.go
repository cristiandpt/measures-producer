package actor

import(
	"github.com/streadway/amqp"
	"log"
	"sync"
	"os"
)

type RabbitMQActor struct {
	queueName       string
	addr            string
	conn            *amqp.Connection
	channel         *amqp.Channel
	notifyConnClose chan *amqp.Error
	notifyChanClose chan *amqp.Error
	isReady         bool
	mailbox         chan interface{} // Actor's mailbox for messages
	wg              sync.WaitGroup
	logger          *log.Logger
}

func NewRabbitMQActor(queueName, addr string) *RabbitMQActor {
	actor := &RabbitMQActor{
		queueName: queueName,
		addr:      addr,
		mailbox:   make(chan interface{}),
		logger:    log.New(os.Stdout, "[RabbitMQActor] ", log.LstdFlags),
	}
	actor.wg.Add(1)
	go actor.run() // Start the actor's processing loop
	return actor
}
