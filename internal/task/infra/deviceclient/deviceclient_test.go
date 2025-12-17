package deviceclient

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	connect "connectrpc.com/connect"
	devicev1 "github.com/KasumiMercury/primind-central-backend/internal/gen/device/v1"
	devicev1connect "github.com/KasumiMercury/primind-central-backend/internal/gen/device/v1/devicev1connect"
)

type testDeviceService struct {
	authHeaderCh chan string
	getResp      *devicev1.GetUserDevicesResponse
	getErr       error
}

type inMemoryHTTPClient struct {
	handler http.Handler
}

func (c *inMemoryHTTPClient) Do(req *http.Request) (*http.Response, error) {
	rec := httptest.NewRecorder()
	c.handler.ServeHTTP(rec, req)
	res := rec.Result()
	if res.Body == nil {
		res.Body = io.NopCloser(rec.Body)
	}
	return res, nil
}

func (s *testDeviceService) RegisterDevice(context.Context, *devicev1.RegisterDeviceRequest) (*devicev1.RegisterDeviceResponse, error) {
	return &devicev1.RegisterDeviceResponse{DeviceId: "device-id"}, nil
}

func (s *testDeviceService) GetUserDevices(ctx context.Context, _ *devicev1.GetUserDevicesRequest) (*devicev1.GetUserDevicesResponse, error) {
	if callInfo, ok := connect.CallInfoForHandlerContext(ctx); ok && s.authHeaderCh != nil {
		s.authHeaderCh <- callInfo.RequestHeader().Get("Authorization")
	}
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.getResp, nil
}

func TestDeviceClientGetUserDevices_SendsAuthorizationHeader(t *testing.T) {
	authHeaderCh := make(chan string, 1)
	wantAuthHeader := "Bearer token-123"
	wantDeviceID := "device-1"

	svc := &testDeviceService{
		authHeaderCh: authHeaderCh,
		getResp: &devicev1.GetUserDevicesResponse{
			Devices: []*devicev1.DeviceInfo{
				{DeviceId: wantDeviceID},
			},
		},
	}
	path, handler := devicev1connect.NewDeviceServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, handler)

	httpClient := &inMemoryHTTPClient{handler: mux}
	client := NewDeviceClientWithHTTPClient("http://example", httpClient)

	devices, err := client.GetUserDevices(context.Background(), "token-123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(devices) != 1 || devices[0].DeviceID != wantDeviceID {
		t.Fatalf("unexpected devices: %#v", devices)
	}

	select {
	case got := <-authHeaderCh:
		if got != wantAuthHeader {
			t.Fatalf("expected Authorization header %q, got %q", wantAuthHeader, got)
		}
	default:
		t.Fatalf("expected Authorization header to be captured")
	}
}

func TestDeviceClientGetUserDevices_MapsUnauthenticatedToErrUnauthorized(t *testing.T) {
	authHeaderCh := make(chan string, 1)

	svc := &testDeviceService{
		authHeaderCh: authHeaderCh,
		getErr:       connect.NewError(connect.CodeUnauthenticated, errors.New("unauthenticated")),
	}
	path, handler := devicev1connect.NewDeviceServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, handler)

	httpClient := &inMemoryHTTPClient{handler: mux}
	client := NewDeviceClientWithHTTPClient("http://example", httpClient)

	_, err := client.GetUserDevices(context.Background(), "token-456")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected %v, got %v", ErrUnauthorized, err)
	}

	select {
	case got := <-authHeaderCh:
		if got != "Bearer token-456" {
			t.Fatalf("expected Authorization header %q, got %q", "Bearer token-456", got)
		}
	default:
		t.Fatalf("expected Authorization header to be captured")
	}
}
