package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	eventcounter "github.com/reb-felipe/eventcounter/pkg"
)

const (
	queueName       = "eventcountertest"
	consumerTag     = "eventcounter-consumer"
	shutdownTimeout = 5 * time.Second 
)

type MessagePayload struct {
	ID string `json:"id"`
}

type RabbitMQConsumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	service eventcounter.Consumer 
}

func NewRabbitMQConsumer(url, exchangeName string, service eventcounter.Consumer) (*RabbitMQConsumer, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("falha ao conectar ao RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("falha ao abrir um canal: %w", err)
	}

	

	return &RabbitMQConsumer{
		conn:    conn,
		channel: ch,
		service: service,
	}, nil
}

// Consume inicia o loop de consumo de mensagens.
func (c *RabbitMQConsumer) Consume(ctx context.Context) error {
	msgs, err := c.channel.Consume(
		queueName,   // queue
		consumerTag, // consumer
		false,       // auto-ack (false para ack manual)
		false,       // exclusive
		false,       // no-local
		false,       // no-wait
		nil,         // args
	)
	if err != nil {
		return fmt.Errorf("falha ao registrar o consumidor: %w", err)
	}

	log.Println("Loop de consumo iniciado.")
	shutdownTimer := time.NewTimer(shutdownTimeout)

	for {
		select {
		case <-ctx.Done():
			log.Println("Contexto cancelado. Encerrando o consumo...")
			return ctx.Err()

		case <-shutdownTimer.C:
			log.Printf("Nenhuma mensagem recebida por %s. Desligando...", shutdownTimeout)
			return nil

		case d, ok := <-msgs:
			if !ok {
				log.Println("Canal de mensagens fechado pelo RabbitMQ.")
				return nil
			}

			if !shutdownTimer.Stop() {
				<-shutdownTimer.C
			}
			shutdownTimer.Reset(shutdownTimeout)

			
			c.processMessage(ctx, d)
		}
	}
}

func (c *RabbitMQConsumer) processMessage(ctx context.Context, d amqp.Delivery) {
	// Extrai informações da routing key.
	parts := strings.Split(d.RoutingKey, ".")
	if len(parts) != 3 || parts[1] != "event" {
		log.Printf("Routing key inválida, descartando mensagem: %s", d.RoutingKey)
		d.Nack(false, false) 
		return
	}
	userID := parts[0]
	eventType := parts[2]

	// Decodifica o corpo da mensagem para obter o ID da mensagem.
	var payload MessagePayload
	if err := json.Unmarshal(d.Body, &payload); err != nil {
		log.Printf("Erro ao decodificar JSON, descartando mensagem: %v", err)
		d.Nack(false, false)
		return
	}

	msgCtx := context.WithValue(ctx, "messageID", payload.ID)

	var err error
	switch eventcounter.EventType(eventType) {
	case eventcounter.EventCreated:
		err = c.service.Created(msgCtx, userID)
	case eventcounter.EventUpdated:
		err = c.service.Updated(msgCtx, userID)
	case eventcounter.EventDeleted:
		err = c.service.Deleted(msgCtx, userID)
	default:
		log.Printf("Tipo de evento desconhecido: %s", eventType)
		d.Nack(false, false) 
		return
	}

	if err != nil {
		log.Printf("Mensagem para usuário '%s' não processada: %v", userID, err)
	}

	d.Ack(false)
}

func (c *RabbitMQConsumer) Close() {
	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
	log.Println("Conexão com RabbitMQ fechada.")
}
