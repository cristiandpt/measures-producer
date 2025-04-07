package actor

import (
	model "github.com/cristiandpt/measures-producer/internal/model"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
	"os"
	"sync"
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

const (
	reconnectDelay = 5 * time.Second
	reInitDelay    = 2 * time.Second
	resendDelay    = 5 * time.Second
)

var (
	errNotConnected = errors.New("not connected to a server")
)

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

func (actor *RabbitMQActor) run() {
	defer actor.wg.Done()
	actor.handleReconnect()

	for msg := range actor.mailbox {
		switch m := msg.(type) {
		case model.PushMessage:
			actor.handlePush(m.Data)
		case model.CloseMessage:
			actor.handleClose()
			return
		default:
			actor.logger.Printf("Received unknown message type: %T\n", msg)
		}
	}
}

// handlePush attempts to push data to the queue with retry.
func (actor *RabbitMQActor) handlePush(data []byte) {
	if !actor.isReady {
		actor.logger.Println("Not connected, cannot push message.")
		return
	}

	for {
		err  := actor.unsafePush(data)
		if err == nil {
			// Confirmation handling (simplified for actor model example)
			// In a more complex scenario, you might want to track confirmations per message.
			actor.logger.Printf("Message pushed successfully: %s\n", string(data))
			return
		}
		actor.logger.Printf("Push failed: %s. Retrying in %s...\n", err, resendDelay)
		select {
		case <-time.After(resendDelay):
		case <-actor.mailbox: // Allow exiting if the actor is closed during retry
			return
		}
	}
}

func (actor *RabbitMQActor) handleClose() {
	actor.logger.Println("Closing connection...")
	if actor.channel != nil {
		if err := actor.channel.Close(); err != nil {
			actor.logger.Printf("Error closing channel: %s\n", err)
		}
	}
	if actor.conn != nil {
		if err := actor.conn.Close(); err != nil {
			actor.logger.Printf("Error closing connection: %s\n", err)
		}
	}
	actor.isReady = false
	close(actor.mailbox) // Close the mailbox to signal the run loop to exit
}

// unsafePush publishes the message to the queue without waiting for confirmation.
func (actor *RabbitMQActor) unsafePush(data []byte) error {
	if !actor.isReady || actor.channel == nil {
		return errNotConnected
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return actor.channel.PublishWithContext(
		ctx,
		"",             // exchange
		actor.queueName,    // routing key
		false,          // mandatory
		false,          // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        data,
		},
	)
}

func handleError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

//A new AMQP connection.
func (actor *RabbitMQActor) connect(addr string) (*amqp.Connection, error) {
	conn, err := amqp.Dial(addr)
	if err != nil {
		handleError(err, "Dialing failed to RabbitMQ broker")
		return nil, err
	}
	actor.changeConnection(conn) 
	actor.isReady = true
	actor.logger.Println("Connected to RabbitMQ!")
	return conn, nil
}


// changeConnection takes a new connection and updates the close listener.
func (actor *RabbitMQActor) changeConnection(connection *amqp.Connection) {
	actor.conn = connection
	actor.notifyConnClose = make(chan *amqp.Error, 1)
	actor.conn.NotifyClose(a.notifyConnClose)
}


// handleReconnect will wait for a connection error and continuously attempt to reconnect.
func (actor *RabbitMQActor) handleReconnect() {
	for {
		actor.isReady = false
		actor.logger.Println("Attempting to connect...")

		conn, err := actor.connect(actor.addr)
		if err != nil {
			actor.logger.Printf("Failed to connect: %s. Retrying in %s...\n", err, reconnectDelay)
			select {
			case <-time.After(reconnectDelay):
			case <-actor.mailbox: // Allow exiting if the actor is closed during reconnect
				return
			}
			continue
		}

		if actor.handleReInit(conn) {
			return // Exit if re-initialization was part of a shutdown
		}
	}
}

// handleReInit will wait for a channel error and continuously attempt to re-initialize the channel.
func (actor *RabbitMQActor) handleReInit(conn *amqp.Connection) bool {
	for {
		if err := actor.init(conn); err != nil {
			actor.logger.Printf("Failed to initialize channel: %s. Retrying in %s...\n", err, reInitDelay)
			select {
			case <-time.After(reInitDelay):
			case <-actor.notifyConnClose:
				actor.logger.Println("Connection closed. Reconnecting...")
				return false
			case <-actor.mailbox: // Allow exiting if the actor is closed during re-init
				return true
			}
			continue
		}

		select {
		case <-actor.mailbox: // Allow exiting if the actor is closed
			return true
		case <-actor.notifyConnClose:
			actor.logger.Println("Connection closed. Reconnecting...")
			return false
		case <-actor.notifyChanClose:
			actor.logger.Println("Channel closed. Re-initializing...")
		}
	}
}

// init will initialize the channel and declare the queue.
func (actor *RabbitMQActor) init(conn *amqp.Connection) error {
	ch, err := conn.Channel()
	if err != nil {
		return err
	}

	if err := ch.Confirm(false); err != nil {
		return err
	}

	_, err = ch.QueueDeclare(
		actor.queueName, // name
		false,       // durable
		false,       // delete when unused
		false,       // exclusive
		false,       // no-wait
		nil,         // arguments
	)
	if err != nil {
		return err
	}

	actor.changeChannel(ch)
	actor.logger.Println("Channel initialized and queue declared.")
	return nil
}

// changeChannel takes a new channel and updates the channel listeners.
func (actor *RabbitMQActor) changeChannel(channel *amqp.Channel) {
	actor.channel = channel
	actor.notifyChanClose = make(chan *amqp.Error, 1)
	// a.notifyConfirm = make(chan amqp.Confirmation, 1) // We'll handle confirms in Push
	actor.channel.NotifyClose(actor.notifyChanClose)
	// a.channel.NotifyPublish(a.notifyConfirm)
}

func (actor *RabbitMQActor) Push(data []byte) {
	actor.mailbox <- PushMessage{Data: data}
}

func (actor *RabbitMQActor) Close() {
	actor.mailbox <- CloseMessage{}
	actor.wg.Wait() // Wait for the actor to finish
	actor.logger.Println("Actor stopped.")
}

func (actor *RabbitMQActor) handleClose() {
	actor.logger.Println("Closing connection...")
	if actor.channel != nil {
		if err := actor.channel.Close(); err != nil {
			actor.logger.Printf("Error closing channel: %s\n", err)
		}
	}
	if actor.conn != nil {
		if err := actor.conn.Close(); err != nil {
			actor.logger.Printf("Error closing connection: %s\n", err)
		}
	}
	actor.isReady = false
	close(actor.mailbox) // Close the mailbox to signal the run loop to exit
}
