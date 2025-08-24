package eventcounter

import (
	"context"
	"log"
)

type LoggingConsumer struct{}

func (LoggingConsumer) Created(ctx context.Context, uid string) error {
	log.Printf("Processed CREATED event for uid=%s", uid)
	return nil
}

func (LoggingConsumer) Updated(ctx context.Context, uid string) error {
	log.Printf("Processed UPDATED event for uid=%s", uid)
	return nil
}

func (LoggingConsumer) Deleted(ctx context.Context, uid string) error {
	log.Printf("Processed DELETED event for uid=%s", uid)
	return nil
}
