// Package main implements sparrow-sinks: an HTTP server that receives signed
// Sparrow webhook deliveries and forwards them to non-HTTP destinations
// (SMTP email, S3). Sinks are stateless: any downstream failure returns a
// non-2xx status so Sparrow's retry machinery redelivers.
package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type smtpConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	StartTLS bool   `yaml:"starttls"`
}

type emailConfig struct {
	WebhookSecret   string     `yaml:"webhook_secret"` // optional per-sink override
	SMTP            smtpConfig `yaml:"smtp"`
	From            string     `yaml:"from"`
	To              []string   `yaml:"to"`
	SubjectTemplate string     `yaml:"subject_template"`
	BodyTemplate    string     `yaml:"body_template"`
}

type s3Config struct {
	WebhookSecret string `yaml:"webhook_secret"` // optional per-sink override
	Endpoint      string `yaml:"endpoint"`       // empty → AWS; set for MinIO/R2
	Region        string `yaml:"region"`
	Bucket        string `yaml:"bucket"`
	Prefix        string `yaml:"prefix"`
}

type config struct {
	Listen        string       `yaml:"listen"`
	WebhookSecret string       `yaml:"webhook_secret"`
	Email         *emailConfig `yaml:"email"`
	S3            *s3Config    `yaml:"s3"`
	OTLP          *otlpConfig  `yaml:"otlp"`
}

func loadConfig(path string) (*config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if c.Listen == "" {
		c.Listen = ":8788"
	}
	configured := 0
	for _, d := range sinkDefs {
		if !d.configured(&c) {
			continue
		}
		configured++
		if err := d.validate(&c); err != nil {
			return nil, err
		}
		if c.secretFor(d.secret(&c)) == "" {
			return nil, fmt.Errorf("%s: webhook_secret is required (top-level or per-sink)", d.name)
		}
	}
	if configured == 0 {
		return nil, fmt.Errorf("%s: no sinks configured (need email:, s3:, and/or otlp:)", path)
	}
	return &c, nil
}

// secretFor returns the per-sink override when set, else the top-level secret.
func (c *config) secretFor(override string) string {
	if override != "" {
		return override
	}
	return c.WebhookSecret
}
