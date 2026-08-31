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
	writer *kafka.Writer
	reader *kafka.Reader
	hub    ConnectionHub
}

type AIResponse struct {
	SessionID string `json:"session_id"`
	Response  string `json:"response"`
}

func NewKafkaService(hub ConnectionHub) *KafkaService {
	brokersEnv := os.Getenv("KAFKA_BROKERS")
	var brokers []string
	if brokersEnv == "" {
		brokers = []string{"localhost:9092"}
	} else {
		brokers = strings.Split(brokersEnv, ",")
	}

	writer := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Balancer: &kafka.LeastBytes{},
	}

	groupID := os.Getenv("KAFKA_GROUP_ID")
	if groupID == "" {
		groupID = "archon-go-gateways"
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		Topic:       "ai.responses",
		GroupID:     groupID,
		StartOffset: kafka.LastOffset,
	})

	return &KafkaService{
		writer: writer,
		reader: reader,
		hub:    hub,
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
			Topic: "ai.requests",
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

func (k *KafkaService) PublishEvent(ctx context.Context, topic string, key []byte, payload []byte) error {
	return k.writer.WriteMessages(ctx,
		kafka.Message{
			Topic: topic,
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
	return k.writer.WriteMessages(context.Background(),
		kafka.Message{
			Topic: "diagram.events",
			Key:   []byte(sessionID),
			Value: bytes,
		},
	)
}
