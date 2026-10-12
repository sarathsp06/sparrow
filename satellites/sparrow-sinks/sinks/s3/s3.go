// Package s3 is the S3 sink: each verified Sparrow delivery is archived as one
// object in an S3-compatible bucket (AWS, MinIO, R2). It is the only sink with
// a heavy dependency (aws-sdk-go-v2), which is why it lives in its own
// package.
package s3

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/sarathsp06/sparrow/satellites/sparrow-sinks/sinks"
)

// Config is the s3: section of sinks.yaml. Credentials come from the
// standard AWS chain (env, shared config, IMDS).
type Config struct {
	WebhookSecret    string `yaml:"webhook_secret"`
	SigningPublicKey string `yaml:"signing_public_key"`
	Endpoint         string `yaml:"endpoint"` // empty → AWS; set for MinIO/R2
	Region           string `yaml:"region"`
	Bucket           string `yaml:"bucket"`
	Prefix           string `yaml:"prefix"`
}

// Handler returns the S3 sink: each verified delivery is archived as one
// object, {prefix}{event_name}/{YYYY}/{MM}/{DD}/{event_id}.json. It verifies
// with SigningPublicKey when set, else WebhookSecret; one of them is
// required. ctx is only used to load the AWS config.
func Handler(ctx context.Context, cfg Config, logger *slog.Logger) (http.Handler, error) {
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("s3: bucket is required")
	}
	verify, err := sinks.VerifierFor(cfg.WebhookSecret, cfg.SigningPublicKey)
	if err != nil {
		return nil, fmt.Errorf("s3: %w", err)
	}
	sink, err := newS3Sink(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return sinks.Handle("s3", verify, logger, sink.deliver), nil
}

type s3Sink struct {
	client *awss3.Client
	bucket string
	prefix string
}

// newS3Sink builds a sink using the standard AWS credential chain
// (env, shared config, IMDS). A non-empty endpoint targets MinIO/R2
// with path-style addressing.
func newS3Sink(ctx context.Context, cfg Config) (*s3Sink, error) {
	var opts []func(*awsconfig.LoadOptions) error
	if cfg.Region != "" {
		opts = append(opts, awsconfig.WithRegion(cfg.Region))
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("s3: load aws config: %w", err)
	}
	client := awss3.NewFromConfig(awsCfg, func(o *awss3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			o.UsePathStyle = true
		}
	})
	return &s3Sink{client: client, bucket: cfg.Bucket, prefix: cfg.Prefix}, nil
}

// objectKey builds {prefix}{event_name}/{YYYY}/{MM}/{DD}/{event_id}.json,
// dated from the envelope timestamp (falling back to now on parse failure).
func objectKey(prefix, eventName, timestamp, eventID string) string {
	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		t = time.Now().UTC()
	}
	return fmt.Sprintf("%s%s/%04d/%02d/%02d/%s.json", prefix, eventName, t.Year(), int(t.Month()), t.Day(), eventID)
}

// deliver archives the raw envelope JSON, one object per delivery.
func (s *s3Sink) deliver(ctx context.Context, env sinks.Envelope, raw []byte) error {
	_, err := s.client.PutObject(ctx, &awss3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(objectKey(s.prefix, env.EventName, env.Timestamp, env.EventID)),
		Body:        bytes.NewReader(raw),
		ContentType: aws.String("application/json"),
	})
	if err != nil {
		return fmt.Errorf("s3 put: %w", err)
	}
	return nil
}
