package sources

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
)

type fakeKafkaFetcher struct {
	messages  []kafka.Message
	committed []kafka.Message
	closed    bool
}

func (f *fakeKafkaFetcher) FetchMessage(ctx context.Context) (kafka.Message, error) {
	if len(f.messages) == 0 {
		<-ctx.Done()
		return kafka.Message{}, ctx.Err()
	}
	msg := f.messages[0]
	f.messages = f.messages[1:]
	return msg, nil
}

func (f *fakeKafkaFetcher) CommitMessages(ctx context.Context, msgs ...kafka.Message) error {
	f.committed = append(f.committed, msgs...)
	return nil
}

func (f *fakeKafkaFetcher) Close() error {
	f.closed = true
	return nil
}

type fakeKafkaPusher struct {
	pushed []struct {
		event   string
		payload json.RawMessage
		labels  map[string]string
	}
}

func (p *fakeKafkaPusher) PushEvent(ctx context.Context, event string, payload json.RawMessage, labels map[string]string) error {
	p.pushed = append(p.pushed, struct {
		event   string
		payload json.RawMessage
		labels  map[string]string
	}{event: event, payload: payload, labels: labels})
	return nil
}

func TestKafkaConfigDefaults(t *testing.T) {
	content := []byte(`
sparrow:
  url: http://localhost:8080
kafka:
  - {}
  - topic: orders
    brokers: ["10.0.0.1:9092"]
    group_id: my-group
    sparrow_event_prefix: custom-kafka
    event: custom.orders
    labels:
      env: prod
`)
	tmpFile, err := os.CreateTemp("", "kafka_test_*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.Write(content)
	require.NoError(t, err)
	tmpFile.Close()

	cfg, err := LoadConfig(tmpFile.Name())
	require.NoError(t, err)
	require.Len(t, cfg.Kafka, 2)

	// Default values check
	k0 := cfg.Kafka[0]
	require.Equal(t, []string{"localhost:9092"}, k0.Brokers)
	require.Equal(t, "events", k0.Topic)
	require.Equal(t, "sparrow-sources", k0.GroupID)
	require.Equal(t, "kafka", k0.SparrowEventPrefix)

	// Custom values check
	k1 := cfg.Kafka[1]
	require.Equal(t, []string{"10.0.0.1:9092"}, k1.Brokers)
	require.Equal(t, "orders", k1.Topic)
	require.Equal(t, "my-group", k1.GroupID)
	require.Equal(t, "custom-kafka", k1.SparrowEventPrefix)
	require.Equal(t, "custom.orders", k1.Event)
	require.Equal(t, "prod", k1.Labels["env"])
}

func TestMapKafkaMessage(t *testing.T) {
	t.Run("JSON payload and default event prefix", func(t *testing.T) {
		cfg := KafkaConfig{
			Topic:              "user-events",
			SparrowEventPrefix: "kafka",
			Labels:             map[string]string{"env": "staging"},
		}
		msg := kafka.Message{
			Topic: "user-events",
			Key:   []byte("user_123"),
			Value: []byte(`{"action":"signup","user_id":123}`),
		}

		event, payload, labels := mapKafkaMessage(cfg, msg)
		require.Equal(t, "kafka.user-events", event)
		require.JSONEq(t, `{"action":"signup","user_id":123}`, string(payload))
		require.Equal(t, "kafka", labels["source"])
		require.Equal(t, "user-events", labels["topic"])
		require.Equal(t, "user_123", labels["key"])
		require.Equal(t, "staging", labels["env"])
	})

	t.Run("Non-JSON payload and event override", func(t *testing.T) {
		cfg := KafkaConfig{
			Topic: "logs",
			Event: "app.log_line",
		}
		msg := kafka.Message{
			Topic: "logs",
			Value: []byte("some raw log string"),
		}

		event, payload, labels := mapKafkaMessage(cfg, msg)
		require.Equal(t, "app.log_line", event)
		require.JSONEq(t, `{"value":"some raw log string"}`, string(payload))
		require.Equal(t, "kafka", labels["source"])
		require.Equal(t, "logs", labels["topic"])
		require.Empty(t, labels["key"])
	})
}

func TestRunKafkaLoop(t *testing.T) {
	cfg := KafkaConfig{
		Topic:              "payments",
		SparrowEventPrefix: "kafka",
	}

	fetcher := &fakeKafkaFetcher{
		messages: []kafka.Message{
			{
				Topic: "payments",
				Key:   []byte("pay_1"),
				Value: []byte(`{"status":"completed"}`),
			},
		},
	}

	pusher := &fakeKafkaPusher{}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		for {
			if len(pusher.pushed) > 0 {
				cancel()
				return
			}
		}
	}()

	runKafkaLoop(ctx, cfg, fetcher, pusher, log)

	require.True(t, fetcher.closed)
	require.Len(t, pusher.pushed, 1)
	require.Equal(t, "kafka.payments", pusher.pushed[0].event)
	require.JSONEq(t, `{"status":"completed"}`, string(pusher.pushed[0].payload))
	require.Len(t, fetcher.committed, 1)
}
