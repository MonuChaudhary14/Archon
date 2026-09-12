package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/MonuChaudhary14/Archon/pkg/telemetry"
	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type KafkaHeaderCarrier []kafka.Header

func (c *KafkaHeaderCarrier) Get(key string) string {
	for _, h := range *c {
		if strings.EqualFold(h.Key, key) {
			return string(h.Value)
		}
	}
	return ""
}

func (c *KafkaHeaderCarrier) Set(key, val string) {
	for i, h := range *c {
		if strings.EqualFold(h.Key, key) {
			(*c)[i].Value = []byte(val)
			return
		}
	}
	*c = append(*c, kafka.Header{Key: key, Value: []byte(val)})
}

func (c *KafkaHeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(*c))
	for _, h := range *c {
		keys = append(keys, h.Key)
	}
	return keys
}

type KafkaService struct {
	writer *kafka.Writer
	reader *kafka.Reader
	hub    ConnectionHub
}

type AIResponse struct {
	SessionID string `json:"session_id"`
	Delta     string `json:"delta,omitempty"`
	Response  string `json:"response,omitempty"`
	IsFinal   bool   `json:"is_final"`
	State     string `json:"state,omitempty"`
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
		Addr:                   kafka.TCP(brokers...),
		Balancer:               &kafka.Murmur2Balancer{},
		AllowAutoTopicCreation: true,
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

func (k *KafkaService) Close() error {
	var errs []string
	if err := k.writer.Close(); err != nil {
		errs = append(errs, err.Error())
	}
	if err := k.reader.Close(); err != nil {
		errs = append(errs, err.Error())
	}
	if len(errs) > 0 {
		return fmt.Errorf("kafka service close errors: %s", strings.Join(errs, "; "))
	}
	return nil
}

func (k *KafkaService) injectTraceHeaders(ctx context.Context) []kafka.Header {
	carrier := KafkaHeaderCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, &carrier)
	return carrier
}

func (k *KafkaService) PublishPrompt(sessionID, prompt string) error {
	ctx, span := telemetry.Tracer().Start(context.Background(), "Kafka.PublishPrompt",
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			attribute.String("messaging.system", "kafka"),
			attribute.String("messaging.destination", "ai.requests"),
			attribute.String("session_id", sessionID),
		),
	)
	defer span.End()

	payload := map[string]string{
		"session_id": sessionID,
		"prompt":     prompt,
	}
	bytes, _ := json.Marshal(payload)
	headers := k.injectTraceHeaders(ctx)

	err := k.writer.WriteMessages(ctx,
		kafka.Message{
			Topic:   "ai.requests",
			Key:     []byte(sessionID),
			Value:   bytes,
			Headers: headers,
		},
	)
	if err != nil {
		span.RecordError(err)
		return err
	}
	return nil
}

func (k *KafkaService) StartConsuming() {
	propagator := otel.GetTextMapPropagator()
	tracer := telemetry.Tracer()

	for {
		msg, err := k.reader.ReadMessage(context.Background())
		if err != nil {
			log.Printf("Kafka read error: %v\n", err)
			continue
		}

		carrier := KafkaHeaderCarrier(msg.Headers)
		ctx := propagator.Extract(context.Background(), &carrier)

		_, span := tracer.Start(ctx, "Kafka.ConsumeAIResponse",
			trace.WithSpanKind(trace.SpanKindConsumer),
			trace.WithAttributes(
				attribute.String("messaging.system", "kafka"),
				attribute.String("messaging.source", "ai.responses"),
				attribute.Int64("messaging.kafka.offset", msg.Offset),
				attribute.Int("messaging.kafka.partition", msg.Partition),
			),
		)

		var resp AIResponse
		err = json.Unmarshal(msg.Value, &resp)
		if err == nil {
			span.SetAttributes(attribute.String("session_id", resp.SessionID))

			if resp.Delta != "" && !resp.IsFinal {
				chunkPayload := map[string]interface{}{
					"type": "chunk",
					"data": map[string]string{
						"role":  "ai",
						"delta": resp.Delta,
					},
				}
				if payloadBytes, marshalErr := json.Marshal(chunkPayload); marshalErr == nil {
					k.hub.SendMessage(resp.SessionID, payloadBytes)
				}
			} else if resp.IsFinal || resp.Response != "" {
				chatPayload := map[string]interface{}{
					"type": "chat",
					"data": map[string]string{
						"role":    "ai",
						"content": resp.Response,
						"state":   resp.State,
					},
				}
				if payloadBytes, marshalErr := json.Marshal(chatPayload); marshalErr == nil {
					k.hub.SendMessage(resp.SessionID, payloadBytes)
				}
			}
		} else {
			span.RecordError(err)
		}
		span.End()
	}
}

func (k *KafkaService) PublishEvent(ctx context.Context, topic string, key []byte, payload []byte) error {
	ctx, span := telemetry.Tracer().Start(ctx, fmt.Sprintf("Kafka.PublishEvent %s", topic),
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			attribute.String("messaging.system", "kafka"),
			attribute.String("messaging.destination", topic),
			attribute.String("messaging.kafka.key", string(key)),
		),
	)
	defer span.End()

	headers := k.injectTraceHeaders(ctx)

	err := k.writer.WriteMessages(ctx,
		kafka.Message{
			Topic:   topic,
			Key:     key,
			Value:   payload,
			Headers: headers,
		},
	)
	if err != nil {
		span.RecordError(err)
		return err
	}
	return nil
}

func (k *KafkaService) PublishDiagramEvent(sessionID, eventType string, data json.RawMessage) error {
	ctx, span := telemetry.Tracer().Start(context.Background(), "Kafka.PublishDiagramEvent",
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			attribute.String("messaging.system", "kafka"),
			attribute.String("messaging.destination", "diagram.events"),
			attribute.String("session_id", sessionID),
			attribute.String("diagram.event_type", eventType),
		),
	)
	defer span.End()

	payload := map[string]interface{}{
		"session_id": sessionID,
		"event_type": eventType,
		"data":       data,
	}
	bytes, err := json.Marshal(payload)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to marshal diagram event payload: %w", err)
	}

	headers := k.injectTraceHeaders(ctx)

	err = k.writer.WriteMessages(ctx,
		kafka.Message{
			Topic:   "diagram.events",
			Key:     []byte(sessionID),
			Value:   bytes,
			Headers: headers,
		},
	)
	if err != nil {
		span.RecordError(err)
		return err
	}
	return nil
}
