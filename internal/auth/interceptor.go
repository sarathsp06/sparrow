package auth

import (
	"context"
	"log/slog"
	"strings"

	"connectrpc.com/connect"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AuthInterceptorConfig configures the auth interceptor behavior.
type AuthInterceptorConfig struct {
	// Enabled controls whether authentication is enforced.
	// When false, DefaultAuthInfo() is injected into every request context.
	Enabled bool

	// Authenticators is the ordered list of authenticators to try.
	// The first one that matches the credential scheme wins.
	Authenticators []Authenticator

	// SkipProcedures is a set of fully-qualified procedure names that
	// bypass authentication (e.g., health checks).
	// Keys are procedure names like "/sparrow.HealthService/Check".
	SkipProcedures map[string]bool

	// Logger for auth events.
	Logger *slog.Logger
}

// NewConnectInterceptor returns a Connect-RPC interceptor that authenticates
// requests and injects AuthInfo into the context.
func NewConnectInterceptor(cfg AuthInterceptorConfig) connect.Interceptor {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			// Skip auth for excluded procedures
			if cfg.SkipProcedures[req.Spec().Procedure] {
				if !cfg.Enabled {
					ctx = NewContext(ctx, DefaultAuthInfo())
				}
				return next(ctx, req)
			}

			// If auth is disabled, inject default auth info and proceed
			if !cfg.Enabled {
				ctx = NewContext(ctx, DefaultAuthInfo())
				return next(ctx, req)
			}

			// Extract Authorization header
			authHeader := req.Header().Get("Authorization")
			if authHeader == "" {
				return nil, connect.NewError(connect.CodeUnauthenticated, errMissingCredential)
			}

			// Parse "Scheme credential" format
			scheme, credential, ok := parseAuthHeader(authHeader)
			if !ok {
				return nil, connect.NewError(connect.CodeUnauthenticated, errMalformedCredential)
			}

			// Try each authenticator
			info, err := authenticate(ctx, cfg.Authenticators, scheme, credential)
			if err != nil {
				logger.WarnContext(ctx, "authentication failed",
					slog.String("procedure", req.Spec().Procedure),
					slog.String("scheme", scheme),
					slog.String("error", err.Error()),
				)
				return nil, connect.NewError(connect.CodeUnauthenticated, err)
			}

			logger.DebugContext(ctx, "authenticated",
				slog.String("procedure", req.Spec().Procedure),
				slog.String("tenant_id", info.TenantID.String()),
				slog.Bool("platform_admin", info.IsPlatformAdmin),
			)

			ctx = NewContext(ctx, info)
			return next(ctx, req)
		}
	})
}

// NewGRPCUnaryInterceptor returns a gRPC unary server interceptor that
// authenticates requests and injects AuthInfo into the context.
func NewGRPCUnaryInterceptor(cfg AuthInterceptorConfig) grpc.UnaryServerInterceptor {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		// Skip auth for excluded procedures
		if cfg.SkipProcedures[info.FullMethod] {
			if !cfg.Enabled {
				ctx = NewContext(ctx, DefaultAuthInfo())
			}
			return handler(ctx, req)
		}

		// If auth is disabled, inject default auth info and proceed
		if !cfg.Enabled {
			ctx = NewContext(ctx, DefaultAuthInfo())
			return handler(ctx, req)
		}

		// Extract authorization from gRPC metadata
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, errMissingCredential.Error())
		}

		authValues := md.Get("authorization")
		if len(authValues) == 0 {
			return nil, status.Error(codes.Unauthenticated, errMissingCredential.Error())
		}

		// Parse "Scheme credential" format
		scheme, credential, ok := parseAuthHeader(authValues[0])
		if !ok {
			return nil, status.Error(codes.Unauthenticated, errMalformedCredential.Error())
		}

		// Try each authenticator
		authInfo, err := authenticate(ctx, cfg.Authenticators, scheme, credential)
		if err != nil {
			logger.WarnContext(ctx, "authentication failed",
				slog.String("method", info.FullMethod),
				slog.String("scheme", scheme),
				slog.String("error", err.Error()),
			)
			return nil, status.Error(codes.Unauthenticated, err.Error())
		}

		logger.DebugContext(ctx, "authenticated",
			slog.String("method", info.FullMethod),
			slog.String("tenant_id", authInfo.TenantID.String()),
			slog.Bool("platform_admin", authInfo.IsPlatformAdmin),
		)

		ctx = NewContext(ctx, authInfo)
		return handler(ctx, req)
	}
}

// ---- Shared helpers ----

// parseAuthHeader splits "Scheme credential" into parts.
// Returns (scheme, credential, ok).
func parseAuthHeader(header string) (string, string, bool) {
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	scheme := parts[0]
	credential := strings.TrimSpace(parts[1])
	if scheme == "" || credential == "" {
		return "", "", false
	}
	return scheme, credential, true
}

// authenticate tries each authenticator whose scheme matches.
// If multiple authenticators handle the same scheme (e.g., JWT and API key
// both use "Bearer"), each is tried in order until one succeeds.
func authenticate(ctx context.Context, authenticators []Authenticator, scheme, credential string) (*AuthInfo, error) {
	var lastErr error
	for _, authn := range authenticators {
		if !strings.EqualFold(authn.Scheme(), scheme) {
			continue
		}
		info, err := authn.Authenticate(ctx, credential)
		if err == nil {
			return info, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, errUnsupportedScheme
}

// Sentinel errors for auth failures.
var (
	errMissingCredential   = &authError{"missing authorization header"}
	errMalformedCredential = &authError{"malformed authorization header: expected 'Scheme credential'"}
	errUnsupportedScheme   = &authError{"unsupported authentication scheme"}
)

// authError is a simple error type for auth failures.
type authError struct {
	msg string
}

func (e *authError) Error() string {
	return e.msg
}
