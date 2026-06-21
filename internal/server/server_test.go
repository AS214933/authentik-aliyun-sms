package server

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/authentik-aliyun-sms/internal/aliyun"
)

func TestSendAcceptsAuthentikDefaultPayload(t *testing.T) {
	var got aliyun.Message
	srv := New(Config{
		AuthToken: "secret",
		Sender: SenderFunc(func(_ context.Context, msg aliyun.Message) error {
			got = msg
			return nil
		}),
	})
	request := httptest.NewRequest(http.MethodPost, "/send", bytes.NewBufferString(`{"From":"authentik","To":"+15551234567","Body":"123456"}`))
	request.Header.Set("Authorization", "Bearer secret")
	response := httptest.NewRecorder()

	srv.Routes().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", response.Code)
	}
	if got.From != "authentik" || got.To != "+15551234567" || got.Body != "123456" {
		t.Fatalf("unexpected message: %#v", got)
	}
}

func TestSendFallsBackToMessageField(t *testing.T) {
	var got aliyun.Message
	srv := New(Config{
		Sender: SenderFunc(func(_ context.Context, msg aliyun.Message) error {
			got = msg
			return nil
		}),
	})
	request := httptest.NewRequest(http.MethodPost, "/send", bytes.NewBufferString(`{"To":"+15551234567","Message":"123456"}`))
	response := httptest.NewRecorder()

	srv.Routes().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", response.Code)
	}
	if got.Body != "123456" {
		t.Fatalf("unexpected body: %q", got.Body)
	}
}

func TestSendRejectsMissingBearerToken(t *testing.T) {
	srv := New(Config{
		AuthToken: "secret",
		Sender:    SenderFunc(func(context.Context, aliyun.Message) error { return nil }),
	})
	request := httptest.NewRequest(http.MethodPost, "/send", bytes.NewBufferString(`{"To":"+15551234567","Body":"123456"}`))
	response := httptest.NewRecorder()

	srv.Routes().ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status: %d", response.Code)
	}
}

func TestSendReturnsBadGatewayWhenProviderFails(t *testing.T) {
	srv := New(Config{
		Sender: SenderFunc(func(context.Context, aliyun.Message) error {
			return errors.New("provider failed")
		}),
	})
	request := httptest.NewRequest(http.MethodPost, "/send", bytes.NewBufferString(`{"To":"+15551234567","Body":"123456"}`))
	response := httptest.NewRecorder()

	srv.Routes().ServeHTTP(response, request)

	if response.Code != http.StatusBadGateway {
		t.Fatalf("unexpected status: %d", response.Code)
	}
}
