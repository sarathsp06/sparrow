package sources

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"
)

type kafkaMessageFetcher interface {
	FetchMessage(ctx context.Context) (kafka.Message, error)
	CommitMessages(ctx context.Context, msgs ...kafka.Message) error
	Close() error
}

// RunKafka launches a consumer loop for each configured Kafka source.
func RunKafka(ctx context.Context, jobs []KafkaConfig, p Pusher, log *slog.Logger) {
	for i := range jobs {
		job := jobs[i]
		go runSingleKafkaConsumer(ctx, job, p, log)
	}
}

func runSingleKafkaConsumer(ctx context.Context, cfg KafkaConfig, p Pusher, log *slog.Logger) {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  cfg.Brokers,
		Topic:    cfg.Topic,
		GroupID:  cfg.GroupID,
		MinBytes: 10,
		MaxBytes: 10e6,
	})
	runKafkaLoop(ctx, cfg, r, p, log)
}

func runKafkaLoop(ctx context.Context, cfg KafkaConfig, fetcher kafkaMessageFetcher, p Pusher, log *slog.Logger) {
	defer fetcher.Close() //nolint:errcheck
	for {
		if ctx.Err() != nil {
			return
		}
		msg, err := fetcher.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || ctx.Err() != nil {
				return
			}
			log.Error("kafka fetch error", "topic", cfg.Topic, "error", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(1 * time.Second):
			}
			continue
		}

		eventName, payload, labels := mapKafkaMessage(cfg, msg)
		for {
			if err := p.PushEvent(ctx, eventName, payload, labels); err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Error("kafka push failed", "topic", cfg.Topic, "event", eventName, "error", err)
				select {
				case <-ctx.Done():
					return
				case <-time.After(1 * time.Second):
				}
				continue
			}
			break
		}

		if ctx.Err() != nil {
			return
		}

		if err := fetcher.CommitMessages(ctx, msg); err != nil {
			log.Error("kafka commit error", "topic", cfg.Topic, "error", err)
		} else {
			log.Info("kafka event pushed", "topic", cfg.Topic, "event", eventName)
		}
	}
}

// mapKafkaMessage maps a Kafka message to a Sparrow event name, payload, and labels.
func mapKafkaMessage(cfg KafkaConfig, msg kafka.Message) (string, json.RawMessage, map[string]string) {
	eventName := cfg.Event
	if eventName == "" {
		prefix := cfg.SparrowEventPrefix
		if prefix == "" {
			prefix = "kafka"
		}
		topic := msg.Topic
		if topic == "" {
			topic = cfg.Topic
		}
		eventName = prefix + "." + topic
	}

	var payload json.RawMessage
	if json.Valid(msg.Value) {
		payload = json.RawMessage(msg.Value)
	} else {
		b, _ := json.Marshal(map[string]string{"value": string(msg.Value)})
		payload = json.RawMessage(b)
	}

	labels := make(map[string]string)
	labels["source"] = "kafka"
	if msg.Topic != "" {
		labels["topic"] = msg.Topic
	} else if cfg.Topic != "" {
		labels["topic"] = cfg.Topic
	}
	if len(msg.Key) > 0 {
		labels["key"] = string(msg.Key)
	}
	for k, v := range cfg.Labels {
		labels[k] = v
	}

	return eventName, payload, labels
}
