package server

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/example/authentik-aliyun-sms/internal/aliyun"
)

type Config struct {
	AuthToken string
	Logger    *slog.Logger
	Sender    aliyun.Sender
}

type Server struct {
	authToken string
	logger    *slog.Logger
	sender    aliyun.Sender
}

type smsRequest struct {
	From    string `json:"From"`
	To      string `json:"To"`
	Body    string `json:"Body"`
	Message string `json:"Message"`
}

type response struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

func New(cfg Config) *Server {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{
		authToken: strings.TrimSpace(cfg.AuthToken),
		logger:    logger,
		sender:    cfg.Sender,
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("POST /send", s.send)
	return mux
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, response{Status: "ok"})
}

func (s *Server) send(w http.ResponseWriter, r *http.Request) {
	if s.authToken != "" && !validBearer(r.Header.Get("Authorization"), s.authToken) {
		writeJSON(w, http.StatusUnauthorized, response{Status: "error", Error: "unauthorized"})
		return
	}

	defer r.Body.Close()
	var payload smsRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, response{Status: "error", Error: "invalid json body"})
		return
	}

	body := strings.TrimSpace(payload.Body)
	if body == "" {
		body = strings.TrimSpace(payload.Message)
	}

	msg := aliyun.Message{
		From: payload.From,
		To:   payload.To,
		Body: body,
	}
	if err := validateMessage(msg); err != nil {
		writeJSON(w, http.StatusBadRequest, response{Status: "error", Error: err.Error()})
		return
	}

	if err := s.sender.Send(r.Context(), msg); err != nil {
		s.logger.Error("sms send failed", "error", err, "to", payload.To)
		writeJSON(w, http.StatusBadGateway, response{Status: "error", Error: "sms provider failed"})
		return
	}

	writeJSON(w, http.StatusOK, response{Status: "ok"})
}

func validateMessage(msg aliyun.Message) error {
	if strings.TrimSpace(msg.To) == "" {
		return errors.New("To is required")
	}
	if strings.TrimSpace(msg.Body) == "" {
		return errors.New("Body or Message is required")
	}
	return nil
}

func validBearer(header, token string) bool {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return false
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix)) == token
}

func writeJSON(w http.ResponseWriter, status int, payload response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

type SenderFunc func(ctx context.Context, msg aliyun.Message) error

func (f SenderFunc) Send(ctx context.Context, msg aliyun.Message) error {
	return f(ctx, msg)
}
