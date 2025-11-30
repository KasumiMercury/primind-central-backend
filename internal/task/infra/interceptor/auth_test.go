package interceptor

import (
	"context"
	"errors"
	"testing"

	connect "connectrpc.com/connect"
)

func TestAuthInterceptorSuccess(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		headerValue   string
		expectedToken string
	}{
		{
			name:          "with bearer prefix",
			headerValue:   "Bearer session-token",
			expectedToken: "session-token",
		},
		{
			name:          "without prefix",
			headerValue:   "raw-token",
			expectedToken: "raw-token",
		},
		{
			name:          "with whitespace and lowercase prefix",
			headerValue:   "   bearer   spaced-token  ",
			expectedToken: "spaced-token",
		},
	}

	for _, tt := range testCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := connect.NewRequest(&struct{}{})
			if tt.headerValue != "" {
				req.Header().Set(tokenHeader, tt.headerValue)
			}

			interceptor := AuthInterceptor()
			var called bool

			next := func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
				called = true

				if got := ExtractSessionToken(ctx); got != tt.expectedToken {
					t.Fatalf("ExtractSessionToken() = %q, want %q", got, tt.expectedToken)
				}

				return connect.NewResponse(&struct{}{}), nil
			}

			if _, err := interceptor(next)(context.Background(), req); err != nil {
				t.Fatalf("AuthInterceptor returned error: %v", err)
			}

			if !called {
				t.Fatalf("next handler was not called")
			}
		})
	}
}

func TestAuthInterceptorError(t *testing.T) {
	t.Parallel()

	errNext := errors.New("next handler error")

	testCases := []struct {
		name          string
		headerValue   string
		expectedToken string
		callNext      bool
	}{
		{
			name:          "propagates error with token",
			headerValue:   "Bearer failure-token",
			expectedToken: "failure-token",
			callNext:      true,
		},
		{
			name:          "propagates error without token",
			headerValue:   "",
			expectedToken: "",
			callNext:      false,
		},
	}

	for _, tt := range testCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := connect.NewRequest(&struct{}{})
			if tt.headerValue != "" {
				req.Header().Set(tokenHeader, tt.headerValue)
			}

			interceptor := AuthInterceptor()
			var called bool

			next := func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
				called = true

				if got := ExtractSessionToken(ctx); got != tt.expectedToken {
					t.Fatalf("ExtractSessionToken() = %q, want %q", got, tt.expectedToken)
				}

				return nil, errNext
			}

			if tt.callNext {
				if _, err := interceptor(next)(context.Background(), req); !errors.Is(err, errNext) {
					t.Fatalf("expected error %v, got %v", errNext, err)
				}

				if !called {
					t.Fatalf("next handler was not called")
				}
			} else {
				if _, err := interceptor(next)(context.Background(), req); !errors.Is(err, ErrTokenHeaderNotFound) {
					t.Fatalf("expected error %v, got %v", ErrTokenHeaderNotFound, err)
				}
			}
		})
	}
}
