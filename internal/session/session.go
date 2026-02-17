package session

import (
	"sync"

	"go.mau.fi/whatsmeow"
)

type Session struct {
	ID        string
	Token     string
	Client    *whatsmeow.Client
	QR        string
	LoggedIn  bool
	Connected bool
	JID       string
	Messages  []IncomingMessage
	Mutex     sync.RWMutex
}

type SessionInfo struct {
	ID        string
	Connected bool
	JID       string
}

type IncomingMessage struct {
	From      string `json:"from"`
	Name      string `json:"name"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

func (s *Session) Snapshot() SessionInfo {
	s.Mutex.RLock()
	defer s.Mutex.RUnlock()

	return SessionInfo{
		ID:        s.ID,
		Connected: s.Connected,
		JID:       s.JID,
	}
}

func (s *Session) SetQR(qr string) {
	s.Mutex.Lock()
	s.QR = qr
	s.Mutex.Unlock()
}

func (s *Session) GetQR() string {
	s.Mutex.RLock()
	defer s.Mutex.RUnlock()
	return s.QR
}

func (s *Session) UpdateStatusFromClient() {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	if s.Client == nil {
		s.Connected = false
		s.LoggedIn = false
		s.JID = ""
		return
	}

	s.Connected = s.Client.IsConnected()
	s.LoggedIn = s.Client.Store.ID != nil
	if s.Client.Store.ID != nil {
		s.JID = s.Client.Store.ID.String()
	} else {
		s.JID = ""
	}
}

func (s *Session) SetConnected(connected bool) {
	s.Mutex.Lock()
	s.Connected = connected
	s.Mutex.Unlock()
}

func (s *Session) SetLoggedIn(loggedIn bool) {
	s.Mutex.Lock()
	s.LoggedIn = loggedIn
	s.Mutex.Unlock()
}

func (s *Session) SetJID(jid string) {
	s.Mutex.Lock()
	s.JID = jid
	s.Mutex.Unlock()
}

func (s *Session) AddMessage(msg IncomingMessage) {
	s.Mutex.Lock()
	s.Messages = append(s.Messages, msg)
	s.Mutex.Unlock()
}

func (s *Session) PopMessages(limit int) []IncomingMessage {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	if limit <= 0 || limit > len(s.Messages) {
		limit = len(s.Messages)
	}
	if limit == 0 {
		return nil
	}

	out := make([]IncomingMessage, limit)
	copy(out, s.Messages[:limit])
	s.Messages = s.Messages[limit:]
	return out
}

func (s *Session) SetToken(token string) {
	s.Mutex.Lock()
	s.Token = token
	s.Mutex.Unlock()
}

func (s *Session) GetToken() string {
	s.Mutex.RLock()
	defer s.Mutex.RUnlock()
	return s.Token
}
