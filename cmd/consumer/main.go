package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/reb-felipe/eventcounter/internal/consumer"
	"github.com/reb-felipe/eventcounter/internal/service"
)

const (
	amqpURL      = "amqp://guest:guest@localhost:5672"
	amqpExchange = "user-events" // O exchange que seu gerador usa
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-shutdownChan
		log.Println("Sinal de desligamento recebido, iniciando o processo...")
		cancel() 
	}()

	eventService := service.NewEventService()

	rabbitConsumer, err := consumer.NewRabbitMQConsumer(amqpURL, amqpExchange, eventService)
	if err != nil {
		log.Fatalf("Falha ao criar o consumidor RabbitMQ: %v", err)
	}
	defer rabbitConsumer.Close()

	log.Println("Consumidor iniciado. Aguardando mensagens...")

	
	if err := rabbitConsumer.Consume(ctx); err != nil {
		if err != context.Canceled {
			log.Printf("Erro durante o consumo de mensagens: %v", err)
		}
	}

	log.Println("Gerando relatórios de contagem...")
	if err := eventService.ShutdownAndWriteResults(); err != nil {
		log.Fatalf("Falha ao escrever resultados em JSON: %v", err)
	}

	log.Println("Processo finalizado com sucesso.")
}