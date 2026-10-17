package interceptor

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect/v2"
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

			info := &connect.CallInfo{}
			if tt.headerValue != "" {
				info.RequestHeader().Set(tokenHeader, tt.headerValue)
			}

			server := connect.NewServer(AuthInterceptor())

			var called bool

			server.SetUnknownHandler(func(ctx context.Context, _ connect.Spec, _ connect.ServerStream) error {
				called = true

				if got := ExtractSessionToken(ctx); got != tt.expectedToken {
					t.Fatalf("ExtractSessionToken() = %q, want %q", got, tt.expectedToken)
				}

				return nil
			})

			if err := server.Call(context.Background(), "/test.v1.TestService/Test", info, nil); err != nil {
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

			info := &connect.CallInfo{}
			if tt.headerValue != "" {
				info.RequestHeader().Set(tokenHeader, tt.headerValue)
			}

			server := connect.NewServer(AuthInterceptor())

			var called bool

			server.SetUnknownHandler(func(ctx context.Context, _ connect.Spec, _ connect.ServerStream) error {
				called = true

				if got := ExtractSessionToken(ctx); got != tt.expectedToken {
					t.Fatalf("ExtractSessionToken() = %q, want %q", got, tt.expectedToken)
				}

				return errNext
			})

			if tt.callNext {
				if err := server.Call(context.Background(), "/test.v1.TestService/Test", info, nil); !errors.Is(err, errNext) {
					t.Fatalf("expected error %v, got %v", errNext, err)
				}

				if !called {
					t.Fatalf("next handler was not called")
				}
			} else {
				if err := server.Call(context.Background(), "/test.v1.TestService/Test", info, nil); !errors.Is(err, ErrTokenHeaderNotFound) {
					t.Fatalf("expected error %v, got %v", ErrTokenHeaderNotFound, err)
				}
			}
		})
	}
}
