package main

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type s3Sink struct {
	client *s3.Client
	bucket string
	prefix string
}

// newS3Sink builds a sink using the standard AWS credential chain
// (env, shared config, IMDS). A non-empty endpoint targets MinIO/R2
// with path-style addressing.
func newS3Sink(ctx context.Context, cfg s3Config) (*s3Sink, error) {
	var opts []func(*awsconfig.LoadOptions) error
	if cfg.Region != "" {
		opts = append(opts, awsconfig.WithRegion(cfg.Region))
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("s3: load aws config: %w", err)
	}
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
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
func (s *s3Sink) deliver(ctx context.Context, env envelope, raw []byte) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
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
