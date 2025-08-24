package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	eventcounter "github.com/reb-felipe/eventcounter/pkg"
)

type UserID string

type EventService struct {
	processedMessages sync.Map
	userCounts        sync.Map
	wg                sync.WaitGroup
	createdChan       chan string
	updatedChan       chan string
	deletedChan       chan string
}

// UserCounter armazena o contador para um usuário e um mutex para protegê-lo.
type UserCounter struct {
	count int
	mu    sync.Mutex
}

// NewEventService cria uma nova instância do serviço de eventos.
func NewEventService() *EventService {
	s := &EventService{
		createdChan: make(chan string, 100), 
		updatedChan: make(chan string, 100),
		deletedChan: make(chan string, 100),
	}

	// Inicia as goroutines consumidoras para cada canal de evento.
	go s.eventProcessor(eventcounter.EventCreated, s.createdChan)
	go s.eventProcessor(eventcounter.EventUpdated, s.updatedChan)
	go s.eventProcessor(eventcounter.EventDeleted, s.deletedChan)

	return s
}

func (s *EventService) eventProcessor(eventType eventcounter.EventType, ch <-chan string) {
	for userID := range ch {
		s.incrementUserCount(eventType, userID)
		s.wg.Done() 
	}
}

func (s *EventService) checkAndStoreMessageID(ctx context.Context) (bool, error) {
	messageID, ok := ctx.Value("messageID").(string)
	if !ok || messageID == "" {
		return false, errors.New("ID da mensagem não encontrado no contexto")
	}

	_, loaded := s.processedMessages.LoadOrStore(messageID, true)
	return !loaded, nil 
}

func (s *EventService) dispatchEvent(ctx context.Context, uid string, ch chan<- string) error {
	isNew, err := s.checkAndStoreMessageID(ctx)
	if err != nil {
		return err
	}
	if !isNew {
		return errors.New("mensagem duplicada")
	}

	s.wg.Add(1) 
	ch <- uid
	return nil
}

func (s *EventService) Created(ctx context.Context, uid string) error {
	return s.dispatchEvent(ctx, uid, s.createdChan)
}

func (s *EventService) Updated(ctx context.Context, uid string) error {
	return s.dispatchEvent(ctx, uid, s.updatedChan)
}

func (s *EventService) Deleted(ctx context.Context, uid string) error {
	return s.dispatchEvent(ctx, uid, s.deletedChan)
}

func (s *EventService) incrementUserCount(eventType eventcounter.EventType, userID string) {
	counters, _ := s.userCounts.LoadOrStore(eventType, &sync.Map{}) 
	userCounters := counters.(*sync.Map)

	counter, _ := userCounters.LoadOrStore(UserID(userID), &UserCounter{})
	userCounter := counter.(*UserCounter)

	userCounter.mu.Lock()
	userCounter.count++
	userCounter.mu.Unlock()

	log.Printf("PROCESSED %s event for uid %s", strings.ToUpper(string(eventType)), userID)
}

func (s *EventService) ShutdownAndWriteResults() error {
	close(s.createdChan)
	close(s.updatedChan)
	close(s.deletedChan)

	s.wg.Wait()

	var finalErr error
	s.userCounts.Range(func(key, value interface{}) bool {
		eventType := key.(eventcounter.EventType)
		userCounters := value.(*sync.Map)

		resultMap := make(map[string]int)
		userCounters.Range(func(userKey, userValue interface{}) bool {
			userID := string(userKey.(UserID))
			counter := userValue.(*UserCounter) //UserKey.(UserID)
			resultMap[userID] = counter.count
			return true
		})

		if len(resultMap) == 0 {
			return true
		}

		jsonData, err := json.MarshalIndent(resultMap, "", "  ")
		if err != nil {
			finalErr = fmt.Errorf("erro ao serializar JSON para %s: %w", eventType, err)
			return false
		}

		fileName := fmt.Sprintf("%s.json", eventType)
		err = os.WriteFile(fileName, jsonData, 0644)
		if err != nil {
			finalErr = fmt.Errorf("erro ao escrever arquivo para %s: %w", eventType, err)
			return false
		}
		log.Printf("Resultados para '%s' salvos em %s", eventType, fileName)
		return true
	})

	return finalErr
}