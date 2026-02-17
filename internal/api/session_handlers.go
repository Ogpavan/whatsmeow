package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"wa-mvp-api/internal/session"
)

type createSessionResponse struct {
	Token string `json:"token"`
}

type sessionStatusResponse struct {
	LoggedIn  bool   `json:"logged_in"`
	Connected bool   `json:"connected"`
	JID       string `json:"jid"`
}

type sessionQRResponse struct {
	QR string `json:"qr"`
}

type sessionListItem struct {
	ID        string `json:"id"`
	Connected bool   `json:"connected"`
	JID       string `json:"jid"`
}

type sendMessageRequest struct {
	Phone   string `json:"phone"`
	Message string `json:"message"`
}

type sendMessageResponse struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type receiveMessagesResponse struct {
	Messages []session.IncomingMessage `json:"messages"`
}

func RegisterSessionRoutes(r chi.Router) {
	r.Post("/sessions", handleCreateSession)
	r.Get("/sessions", handleListSessions)
	r.With(authSession).Get("/session/qr", handleGetSessionQR)
	r.With(authSession).Get("/session/status", handleGetSessionStatus)
	r.With(authSession).Post("/session/send", handleSendMessage)
	r.With(authSession).Get("/session/receive", handleReceiveMessages)
}

func handleCreateSession(w http.ResponseWriter, r *http.Request) {
	manager := session.GetManager()
	token, err := manager.CreateSession()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, createSessionResponse{Token: token})
}

func handleListSessions(w http.ResponseWriter, r *http.Request) {
	manager := session.GetManager()
	list := manager.ListSessions()

	resp := make([]sessionListItem, 0, len(list))
	for _, s := range list {
		resp = append(resp, sessionListItem{
			ID:        s.ID,
			Connected: s.Connected,
			JID:       s.JID,
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

func handleGetSessionQR(w http.ResponseWriter, r *http.Request) {
	sess := getSessionFromContext(r)
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	qr, err := session.GetManager().GetQRByToken(sess.GetToken())
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, sessionQRResponse{QR: qr})
}

func handleGetSessionStatus(w http.ResponseWriter, r *http.Request) {
	sess := getSessionFromContext(r)
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	sess.Mutex.RLock()
	resp := sessionStatusResponse{
		LoggedIn:  sess.LoggedIn,
		Connected: sess.Connected,
		JID:       sess.JID,
	}
	sess.Mutex.RUnlock()

	writeJSON(w, http.StatusOK, resp)
}

func handleSendMessage(w http.ResponseWriter, r *http.Request) {
	sess := getSessionFromContext(r)
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req sendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, sendMessageResponse{Status: "error", Error: "invalid json"})
		return
	}

	req.Phone = strings.TrimSpace(req.Phone)
	req.Phone = strings.TrimPrefix(req.Phone, "+")
	req.Message = strings.TrimSpace(req.Message)
	if req.Phone == "" || req.Message == "" {
		writeJSON(w, http.StatusBadRequest, sendMessageResponse{Status: "error", Error: "phone and message are required"})
		return
	}

	if err := session.GetManager().SendTextByToken(r.Context(), sess.GetToken(), req.Phone, req.Message); err != nil {
		writeJSON(w, http.StatusBadRequest, sendMessageResponse{Status: "error", Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, sendMessageResponse{Status: "sent"})
}

func handleReceiveMessages(w http.ResponseWriter, r *http.Request) {
	sess := getSessionFromContext(r)
	if sess == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	limit := 20
	if q := r.URL.Query().Get("limit"); q != "" {
		if v, err := strconv.Atoi(q); err == nil && v > 0 {
			limit = v
		}
	}

	msgs := sess.PopMessages(limit)
	writeJSON(w, http.StatusOK, receiveMessagesResponse{Messages: msgs})
}

func authSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractBearerToken(r)
		if token == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing bearer token"})
			return
		}

		manager := session.GetManager()
		sess, ok := manager.GetSessionByToken(token)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
			return
		}

		ctx := context.WithValue(r.Context(), sessionKey{}, sess)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type sessionKey struct{}

func getSessionFromContext(r *http.Request) *session.Session {
	val := r.Context().Value(sessionKey{})
	if val == nil {
		return nil
	}
	sess, _ := val.(*session.Session)
	return sess
}

func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}
	parts := strings.Fields(auth)
	if len(parts) != 2 {
		return ""
	}
	if strings.ToLower(parts[0]) != "bearer" {
		return ""
	}
	return parts[1]
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
