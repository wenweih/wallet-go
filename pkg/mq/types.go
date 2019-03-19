package mq

import (
  "github.com/streadway/amqp"
)

// IMessagingClient Defines our interface for connecting, producing and consuming messages.
type IMessagingClient interface {
	ConnectToBroker(connectionString string)
	Publish(msg []byte, exchangeName, exchangeType, bindingKey, queueName string) error
	Subscribe(exchangeName, exchangeType, queueName, bindingKey, consumerName string, handleFunc func(amqp.Delivery)) error
	Close()
}

// MessagingClient Real implementation, encapsulates a pointer to an amqp.Connection
type MessagingClient struct {
	conn *amqp.Connection
}
