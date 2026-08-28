package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/segmentio/kafka-go"
)

type KafkaService struct {
	writer        *kafka.Writer
	diagramWriter *kafka.Writer
	reader        *kafka.Reader
	hub           *Hub
}

type AIResponse struct {
	SessionID string `json:"session_id"`
	Response  string `json:"response"`
}

func NewKafkaService(hub *Hub) *KafkaService {
	brokersEnv := os.Getenv("KAFKA_BROKERS")
	var brokers []string
	if brokersEnv == "" {
		brokers = []string{"localhost:9092"}
	} else {
		brokers = strings.Split(brokersEnv, ",")
	}

	writer := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Topic:    "ai.requests",
		Balancer: &kafka.LeastBytes{},
	}

	diagramWriter := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Topic:    "diagram.events",
		Balancer: &kafka.LeastBytes{},
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers,
		Topic:   "ai.responses",
	})

	return &KafkaService{
		writer:        writer,
		diagramWriter: diagramWriter,
		reader:        reader,
		hub:           hub,
	}

}

func (k *KafkaService) PublishPrompt(sessionID, prompt string) error {
	payload := map[string]string{
		"session_id": sessionID,
		"prompt":     prompt,
	}
	bytes, _ := json.Marshal(payload)
	return k.writer.WriteMessages(context.Background(),
		kafka.Message{
			Key:   []byte(sessionID),
			Value: bytes,
		},
	)
}

func (k *KafkaService) StartConsuming() {
	for {
		msg, err := k.reader.ReadMessage(context.Background())
		if err != nil {
			log.Printf("Kafka read error: %v\n", err)
			continue
		}

		var resp AIResponse
		if err := json.Unmarshal(msg.Value, &resp); err == nil {
			payload := map[string]string{
				"role":    "ai",
				"content": resp.Response,
			}
			payloadBytes, err := json.Marshal(payload)
			if err == nil {
				k.hub.SendMessage(resp.SessionID, payloadBytes)
			} else {
				log.Printf("Failed to marshal live response JSON: %v\n", err)
			}
		}
	}
}

func (k *KafkaService) PublishEvent(ctx context.Context, key []byte, payload []byte) error {
	return k.writer.WriteMessages(ctx,
		kafka.Message{
			Key:   key,
			Value: payload,
		},
	)
}

func (k *KafkaService) PublishDiagramEvent(sessionID, eventType string, data json.RawMessage) error {
	payload := map[string]interface{}{
		"session_id": sessionID,
		"event_type": eventType,
		"data":       data,
	}
	bytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal diagram event payload: %w", err)
	}
	return k.diagramWriter.WriteMessages(context.Background(),
		kafka.Message{
			Key:   []byte(sessionID),
			Value: bytes,
		},
	)
}
