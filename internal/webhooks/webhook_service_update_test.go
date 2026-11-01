package webhooks

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	cryptosvc "github.com/sarathsp06/sparrow/internal/crypto"
	svcerrors "github.com/sarathsp06/sparrow/internal/errors"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

func (m *mockRepo) ReplaceWebhookSubscriptions(ctx context.Context, tenantID uuid.UUID, webhookID uuid.UUID, consumer string, subs []*store.EventSubscription) error {
	return m.Called(ctx, tenantID, webhookID, consumer, subs).Error(0)
}

func newCryptoService(t *testing.T, repo *mockRepo) *WebhookService {
	t.Helper()
	c, err := cryptosvc.NewService(bytes.Repeat([]byte{7}, 32))
	require.NoError(t, err)
	return NewWebhookService(nil, repo, c)
}

func storedWebhook(id uuid.UUID) *store.WebhookRegistration {
	return &store.WebhookRegistration{
		ID:                    id,
		Consumer:              "default",
		URL:                   "https://example.com/hook",
		Active:                true,
		MaxRetries:            3,
		RetryBackoffSeconds:   60,
		RequestTimeoutSeconds: 30,
		ExpectedStatusCodes:   pq.Int64Array{200},
		ContentType:           "application/json",
		SignatureType:         store.SignatureTypeHMAC,
	}
}

// TestUpdateWebhookConfig_LegacyAppliesEveryField covers the no-mask path:
// every non-zero field of the request lands on the stored webhook, events
// are replaced, and a rate limit keeps its limiter state.
func TestUpdateWebhookConfig_LegacyAppliesEveryField(t *testing.T) {
	repo := new(mockRepo)
	svc := newCryptoService(t, repo)
	id := uuid.New()
	repo.On("GetWebhookByID", mock.Anything, mock.Anything, id, "default").Return(storedWebhook(id), nil)
	repo.On("ReplaceWebhookSubscriptions", mock.Anything, mock.Anything, id, "default", mock.MatchedBy(func(subs []*store.EventSubscription) bool {
		return len(subs) == 2 && subs[0].EventName == "order.created" && subs[1].EventName == "order.paid"
	})).Return(nil)
	var saved *store.WebhookRegistration
	repo.On("UpdateWebhook", mock.Anything, mock.Anything, mock.Anything).Run(func(a mock.Arguments) {
		saved = a.Get(2).(*store.WebhookRegistration)
	}).Return(nil)
	repo.On("UpsertRateLimitState", mock.Anything, id).Return(errors.New("ignored: only logged"))

	rps := 5.0
	err := svc.UpdateWebhookConfig(context.Background(), id.String(), "default",
		[]string{"order.created", "order.paid"}, "  https://example.org/new  ", map[string]string{"X-Team": "ops"},
		false, "new description", &WebhookHTTPConfig{
			MaxRetries:            new(7),
			RetryBackoffSeconds:   120,
			RequestTimeoutSeconds: 10,
			ExpectedStatusCodes:   IntArray{200, 202},
			WebhookSecret:         "whsec_new",
			UserAgent:             "Custom/2",
			ContentType:           "application/cloudevents+json",
			CaptureResponseBody:   new(true),
			FollowRedirects:       new(false),
			VerifySSL:             new(false),
			RateLimitRPS:          &rps,
		}, nil, "", false, nil)
	require.NoError(t, err)
	require.NotNil(t, saved)

	assert.Equal(t, "https://example.org/new", saved.URL)
	assert.Equal(t, map[string]string{"X-Team": "ops"}, map[string]string(saved.Headers))
	assert.False(t, saved.Active)
	assert.Equal(t, "new description", saved.Description)
	assert.Equal(t, 7, saved.MaxRetries)
	assert.Equal(t, 120, saved.RetryBackoffSeconds)
	assert.Equal(t, 10, saved.RequestTimeoutSeconds)
	assert.Equal(t, pq.Int64Array{200, 202}, saved.ExpectedStatusCodes)
	assert.Equal(t, "Custom/2", saved.UserAgent)
	assert.Equal(t, "application/cloudevents+json", saved.ContentType)
	assert.True(t, saved.CaptureResponseBody)
	assert.False(t, saved.FollowRedirects)
	assert.False(t, saved.VerifySSL)
	require.NotNil(t, saved.RateLimitRPS)
	assert.InDelta(t, 5.0, *saved.RateLimitRPS, 0)
	secret, err := svc.DecryptWebhookSecret(saved.WebhookSecret)
	require.NoError(t, err)
	assert.Equal(t, "whsec_new", secret)
	repo.AssertExpectations(t)
}

// TestUpdateWebhookConfig_MaskLimitsTheChange checks that with a mask only
// the named fields change: a secret and a rate limit in http_config are
// ignored unless their own paths are in the mask.
func TestUpdateWebhookConfig_MaskLimitsTheChange(t *testing.T) {
	repo := new(mockRepo)
	svc := newCryptoService(t, repo)
	id := uuid.New()
	before := storedWebhook(id)
	repo.On("GetWebhookByID", mock.Anything, mock.Anything, id, "default").Return(before, nil)
	var saved *store.WebhookRegistration
	repo.On("UpdateWebhook", mock.Anything, mock.Anything, mock.Anything).Run(func(a mock.Arguments) {
		saved = a.Get(2).(*store.WebhookRegistration)
	}).Return(nil)
	repo.On("DeleteRateLimitState", mock.Anything, id).Return(errors.New("ignored: only logged"))

	rps := 9.0
	err := svc.UpdateWebhookConfig(context.Background(), id.String(), "default",
		[]string{"order.created"}, "https://example.org/other", nil, false, "changed", &WebhookHTTPConfig{
			UserAgent: "Masked/1", WebhookSecret: "whsec_ignored", RateLimitRPS: &rps,
		}, nil, "", false, []string{"http_config"})
	require.NoError(t, err)

	assert.Equal(t, "https://example.com/hook", saved.URL, "url is not in the mask")
	assert.True(t, saved.Active, "active is not in the mask")
	assert.Empty(t, saved.Description)
	assert.Equal(t, "Masked/1", saved.UserAgent)
	assert.Nil(t, saved.WebhookSecret, "http_config.webhook_secret is not in the mask")
	assert.Nil(t, saved.RateLimitRPS, "http_config.rate_limit_rps is not in the mask")
	repo.AssertNotCalled(t, "ReplaceWebhookSubscriptions", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestUpdateWebhookConfig_SignatureTypeSwitch(t *testing.T) {
	id := uuid.New()
	setup := func(t *testing.T, current store.SignatureType) (*WebhookService, **store.WebhookRegistration) {
		repo := new(mockRepo)
		svc := newCryptoService(t, repo)
		wh := storedWebhook(id)
		wh.SignatureType = current
		if current == store.SignatureTypeEd25519 {
			wh.Ed25519PrivateKey = []byte("old-key")
		}
		repo.On("GetWebhookByID", mock.Anything, mock.Anything, id, "default").Return(wh, nil)
		var saved *store.WebhookRegistration
		repo.On("UpdateWebhook", mock.Anything, mock.Anything, mock.Anything).Run(func(a mock.Arguments) {
			saved = a.Get(2).(*store.WebhookRegistration)
		}).Return(nil)
		repo.On("DeleteRateLimitState", mock.Anything, id).Return(nil)
		return svc, &saved
	}

	t.Run("to ed25519 generates a key", func(t *testing.T) {
		svc, saved := setup(t, store.SignatureTypeHMAC)
		require.NoError(t, svc.UpdateWebhookConfig(context.Background(), id.String(), "default", nil, "", nil, true, "", nil, nil, "ed25519", false, []string{"signature_type"}))
		assert.Equal(t, store.SignatureTypeEd25519, (*saved).SignatureType)
		assert.Len(t, svc.WebhookSigningPublicKeyHex((*saved).Ed25519PrivateKey), 64)
	})
	t.Run("back to hmac drops the key", func(t *testing.T) {
		svc, saved := setup(t, store.SignatureTypeEd25519)
		require.NoError(t, svc.UpdateWebhookConfig(context.Background(), id.String(), "default", nil, "", nil, true, "", nil, nil, "hmac", false, []string{"signature_type"}))
		assert.Equal(t, store.SignatureTypeHMAC, (*saved).SignatureType)
		assert.Nil(t, (*saved).Ed25519PrivateKey)
	})
	t.Run("unknown type is refused", func(t *testing.T) {
		svc, _ := setup(t, store.SignatureTypeHMAC)
		err := svc.UpdateWebhookConfig(context.Background(), id.String(), "default", nil, "", nil, true, "", nil, nil, "rsa", false, []string{"signature_type"})
		requireStatus(t, err, svcerrors.InvalidArgument)
	})
}

func TestUpdateWebhookConfig_SecretHeadersCleared(t *testing.T) {
	repo := new(mockRepo)
	svc := newCryptoService(t, repo)
	id := uuid.New()
	wh := storedWebhook(id)
	enc, err := svc.EncryptSecretHeaders(map[string]string{"Authorization": "Bearer x"})
	require.NoError(t, err)
	wh.SecretHeaders = enc
	repo.On("GetWebhookByID", mock.Anything, mock.Anything, id, "default").Return(wh, nil)
	repo.On("UpdateWebhook", mock.Anything, mock.Anything, mock.MatchedBy(func(w *store.WebhookRegistration) bool {
		return w.SecretHeaders == nil
	})).Return(nil)
	repo.On("DeleteRateLimitState", mock.Anything, id).Return(nil)

	require.NoError(t, svc.UpdateWebhookConfig(context.Background(), id.String(), "default", nil, "", nil, true, "", nil, map[string]string{}, "", false, []string{"secret_headers"}))
	repo.AssertExpectations(t)
}

// Secret header values are never returned, so moving the webhook to another
// origin must not carry them along unless the request sets them again.
func TestUpdateWebhookConfig_SecretHeadersFollowURL(t *testing.T) {
	id := uuid.New()
	ctx := context.Background()
	setup := func(t *testing.T) (*WebhookService, **store.WebhookRegistration) {
		repo := new(mockRepo)
		svc := newCryptoService(t, repo)
		wh := storedWebhook(id)
		enc, err := svc.EncryptSecretHeaders(map[string]string{"X-Api-Key": "k", "X-Other": "o"})
		require.NoError(t, err)
		wh.SecretHeaders = enc
		repo.On("GetWebhookByID", mock.Anything, mock.Anything, id, "default").Return(wh, nil)
		var saved *store.WebhookRegistration
		repo.On("UpdateWebhook", mock.Anything, mock.Anything, mock.Anything).Run(func(a mock.Arguments) {
			saved = a.Get(2).(*store.WebhookRegistration)
		}).Return(nil)
		repo.On("DeleteRateLimitState", mock.Anything, id).Return(nil)
		return svc, &saved
	}
	update := func(svc *WebhookService, url string, secret map[string]string, mask ...string) error {
		return svc.UpdateWebhookConfig(ctx, id.String(), "default", nil, url, nil, true, "", nil, secret, "", false, mask)
	}

	t.Run("another origin without secret_headers is refused", func(t *testing.T) {
		svc, _ := setup(t)
		err := update(svc, "https://example.net/", nil, "url")
		requireStatus(t, err, svcerrors.InvalidArgument)
		assert.Contains(t, err.Error(), "X-Api-Key, X-Other")
	})
	t.Run("a header left out is refused", func(t *testing.T) {
		svc, _ := setup(t)
		err := update(svc, "https://example.org/", map[string]string{"X-Api-Key": "k2"}, "url", "secret_headers")
		requireStatus(t, err, svcerrors.InvalidArgument)
		assert.Contains(t, err.Error(), "X-Other")
	})
	t.Run("every header set or removed is allowed", func(t *testing.T) {
		svc, saved := setup(t)
		require.NoError(t, update(svc, "https://example.org/", map[string]string{"X-Api-Key": "k2", "X-Other": ""}, "url", "secret_headers"))
		got, err := svc.DecryptSecretHeaders((*saved).SecretHeaders)
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"X-Api-Key": "k2"}, got)
	})
	t.Run("an empty object removes them all", func(t *testing.T) {
		svc, saved := setup(t)
		require.NoError(t, update(svc, "https://example.org/", map[string]string{}, "url", "secret_headers"))
		assert.Nil(t, (*saved).SecretHeaders)
	})
	t.Run("same origin keeps them", func(t *testing.T) {
		svc, saved := setup(t)
		require.NoError(t, update(svc, "https://example.com/v2/hook", nil, "url"))
		assert.NotNil(t, (*saved).SecretHeaders)
	})
	t.Run("scheme change is another origin", func(t *testing.T) {
		svc, _ := setup(t)
		requireStatus(t, update(svc, "http://example.com/hook", nil, "url"), svcerrors.InvalidArgument)
	})
}

func TestUpdateWebhookConfig_Rejections(t *testing.T) {
	id := uuid.New()
	ctx := context.Background()

	t.Run("missing ids and bad uuid", func(t *testing.T) {
		svc := NewWebhookService(nil, new(mockRepo), nil)
		requireStatus(t, svc.UpdateWebhookConfig(ctx, "", "default", nil, "", nil, true, "", nil, nil, "", false, nil), svcerrors.InvalidArgument)
		requireStatus(t, svc.UpdateWebhookConfig(ctx, id.String(), "", nil, "", nil, true, "", nil, nil, "", false, nil), svcerrors.InvalidArgument)
		requireStatus(t, svc.UpdateWebhookConfig(ctx, "not-a-uuid", "default", nil, "", nil, true, "", nil, nil, "", false, nil), svcerrors.InvalidArgument)
	})

	t.Run("lookup fails", func(t *testing.T) {
		repo := new(mockRepo)
		repo.On("GetWebhookByID", mock.Anything, mock.Anything, id, "default").Return(nil, errors.New("db down"))
		err := NewWebhookService(nil, repo, nil).UpdateWebhookConfig(ctx, id.String(), "default", nil, "", nil, true, "", nil, nil, "", false, nil)
		require.ErrorContains(t, err, "db down")
	})

	for _, tc := range []struct {
		name          string
		events        []string
		url           string
		headers       map[string]string
		secretHeaders map[string]string
		http          *WebhookHTTPConfig
		mask          []string
		crypto        bool
		want          svcerrors.Status
	}{
		{name: "system event for a normal consumer", events: []string{"sparrow.webhook.disabled"}, mask: []string{"events"}, want: svcerrors.InvalidArgument},
		{name: "blank url", url: "   ", mask: []string{"url"}, want: svcerrors.InvalidArgument},
		{name: "loopback url", url: "http://127.0.0.1/hook", mask: []string{"url"}, want: svcerrors.InvalidArgument},
		{name: "bad header name", headers: map[string]string{"Bad Header": "x"}, mask: []string{"headers"}, want: svcerrors.InvalidArgument},
		{name: "bad secret header name", secretHeaders: map[string]string{"Bad Header": "x"}, mask: []string{"secret_headers"}, want: svcerrors.InvalidArgument},
		{name: "secret without encryption", http: &WebhookHTTPConfig{WebhookSecret: "s"}, mask: []string{"http_config", "http_config.webhook_secret"}, want: svcerrors.FailedPrecondition},
		{name: "secret headers without encryption", secretHeaders: map[string]string{"Authorization": "x"}, mask: []string{"secret_headers"}, want: svcerrors.FailedPrecondition},
		{name: "zero rate limit", http: &WebhookHTTPConfig{RateLimitRPS: new(0.0)}, mask: []string{"http_config", "http_config.rate_limit_rps"}, want: svcerrors.InvalidArgument},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := new(mockRepo)
			repo.On("GetWebhookByID", mock.Anything, mock.Anything, id, "default").Return(storedWebhook(id), nil)
			svc := NewWebhookService(nil, repo, nil)
			err := svc.UpdateWebhookConfig(ctx, id.String(), "default", tc.events, tc.url, tc.headers, true, "", tc.http, tc.secretHeaders, "", false, tc.mask)
			requireStatus(t, err, tc.want)
			repo.AssertNotCalled(t, "UpdateWebhook", mock.Anything, mock.Anything, mock.Anything)
		})
	}

	t.Run("transaction fails", func(t *testing.T) {
		repo := new(mockRepo)
		repo.On("GetWebhookByID", mock.Anything, mock.Anything, id, "default").Return(storedWebhook(id), nil)
		repo.On("ReplaceWebhookSubscriptions", mock.Anything, mock.Anything, id, "default", mock.Anything).Return(errors.New("constraint"))
		err := NewWebhookService(nil, repo, nil).UpdateWebhookConfig(ctx, id.String(), "default", []string{"order.created"}, "", nil, true, "", nil, nil, "", false, []string{"events"})
		require.ErrorContains(t, err, "constraint")
		repo.AssertNotCalled(t, "UpdateWebhook", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("lock fails when turning on requires_transform", func(t *testing.T) {
		repo := new(mockRepo)
		repo.On("GetWebhookByID", mock.Anything, mock.Anything, id, "default").Return(storedWebhook(id), nil)
		repo.On("LockWebhook", mock.Anything, mock.Anything, id, "default", true).Return(nil, errors.New("lock timeout"))
		err := NewWebhookService(nil, repo, nil).UpdateWebhookConfig(ctx, id.String(), "default", nil, "", nil, true, "", nil, nil, "", true, []string{"requires_transform"})
		require.ErrorContains(t, err, "lock timeout")
	})
}

func TestWebhookHTTPConfigValidation(t *testing.T) {
	valid := DefaultWebhookHTTPConfig()
	require.NoError(t, valid.ValidateConfig())

	for name, mutate := range map[string]func(*WebhookHTTPConfig){
		"max_retries too high":    func(c *WebhookHTTPConfig) { c.MaxRetries = new(11) },
		"backoff zero":            func(c *WebhookHTTPConfig) { c.RetryBackoffSeconds = 0 },
		"backoff over an hour":    func(c *WebhookHTTPConfig) { c.RetryBackoffSeconds = 3601 },
		"timeout zero":            func(c *WebhookHTTPConfig) { c.RequestTimeoutSeconds = 0 },
		"no expected codes":       func(c *WebhookHTTPConfig) { c.ExpectedStatusCodes = nil },
		"expected code below 100": func(c *WebhookHTTPConfig) { c.ExpectedStatusCodes = IntArray{99} },
		"expected code above 599": func(c *WebhookHTTPConfig) { c.ExpectedStatusCodes = IntArray{600} },
		"empty content type":      func(c *WebhookHTTPConfig) { c.ContentType = "" },
		"negative rate limit":     func(c *WebhookHTTPConfig) { c.RateLimitRPS = new(-1.0) },
	} {
		t.Run(name, func(t *testing.T) {
			c := DefaultWebhookHTTPConfig()
			mutate(&c)
			requireStatus(t, c.ValidateConfig(), svcerrors.InvalidArgument)
		})
	}

	assert.Equal(t, 4, DerefIntOr(nil, 4))
	assert.Equal(t, 2, DerefIntOr(new(2), 4))
	assert.True(t, DerefBoolOr(nil, true))
	assert.False(t, DerefBoolOr(new(false), true))
}

func TestNormalizePagination(t *testing.T) {
	for _, tc := range []struct{ limit, offset, wantLimit, wantOffset int }{
		{0, 0, 50, 0},
		{-5, -3, 50, 0},
		{maxPageLimit + 1, 10, maxPageLimit, 10},
		{25, 5, 25, 5},
	} {
		l, o := normalizePagination(tc.limit, tc.offset)
		assert.Equal(t, tc.wantLimit, l, "limit for %+v", tc)
		assert.Equal(t, tc.wantOffset, o, "offset for %+v", tc)
	}
}

func TestSchemaValidationErrorText(t *testing.T) {
	bare := &SchemaValidationError{Message: "payload does not match"}
	assert.Equal(t, "payload does not match", bare.Error())
	assert.Equal(t, []string{"payload does not match"}, bare.Warnings())

	detailed := &SchemaValidationError{Message: "payload does not match", Details: map[string]string{
		"":       "missing properties: id",
		"/total": "expected number",
	}}
	msg := detailed.Error()
	assert.True(t, strings.HasPrefix(msg, "payload does not match: "), msg)
	assert.Contains(t, msg, "missing properties: id")
	assert.Contains(t, msg, "/total: expected number")
	assert.ElementsMatch(t, []string{"missing properties: id", "field '/total': expected number"}, detailed.Warnings())
}

// A payload with an error per array item reported every one of them.
func TestSchemaValidationWarningsAreCapped(t *testing.T) {
	details := map[string]string{}
	for i := range 1000 {
		details[fmt.Sprintf("/items/%d", i)] = "expected string"
	}
	e := &SchemaValidationError{Message: "payload does not match", Details: details}
	w := e.Warnings()
	require.Len(t, w, maxSchemaWarnings+1)
	assert.Equal(t, "... and 900 more schema warnings", w[maxSchemaWarnings])
	assert.Equal(t, w, e.Warnings(), "the kept warnings change between calls")
	assert.Contains(t, e.Error(), "... and 900 more schema warnings")
}

func TestSecretHelpersWithoutEncryption(t *testing.T) {
	svc := NewWebhookService(nil, new(mockRepo), nil)

	for _, f := range []func() (any, error){
		func() (any, error) { return svc.EncryptSecretHeaders(nil) },
		func() (any, error) { return svc.DecryptSecretHeaders(nil) },
		func() (any, error) { return svc.EncryptWebhookSecret("") },
		func() (any, error) { return svc.DecryptWebhookSecret(nil) },
		func() (any, error) { return svc.generateEncryptedEd25519Key() },
	} {
		_, err := f()
		require.NoError(t, err, "empty input or no encryption needs no key")
	}

	_, err := svc.EncryptSecretHeaders(map[string]string{"a": "b"})
	requireStatus(t, err, svcerrors.FailedPrecondition)
	_, err = svc.DecryptSecretHeaders([]byte("x"))
	requireStatus(t, err, svcerrors.FailedPrecondition)
	_, err = svc.EncryptWebhookSecret("s")
	requireStatus(t, err, svcerrors.FailedPrecondition)
	_, err = svc.DecryptWebhookSecret([]byte("x"))
	requireStatus(t, err, svcerrors.FailedPrecondition)
	assert.Empty(t, svc.WebhookSigningPublicKeyHex([]byte("x")))

	withKey := newCryptoService(t, new(mockRepo))
	_, err = withKey.DecryptSecretHeaders([]byte("not ciphertext"))
	require.Error(t, err)
	assert.Empty(t, withKey.WebhookSigningPublicKeyHex([]byte("not ciphertext")))
	short, err := withKey.EncryptWebhookSecret("too-short-for-a-key")
	require.NoError(t, err)
	assert.Empty(t, withKey.WebhookSigningPublicKeyHex(short), "a value that is not an Ed25519 key yields no public key")
}
