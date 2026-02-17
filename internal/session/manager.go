package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log"
	"sync"
	"time"

	"go.mau.fi/whatsmeow/types/events"
	"go.mau.fi/whatsmeow/types"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"google.golang.org/protobuf/proto"
	"wa-mvp-api/internal/whatsapp"
)

type Manager struct {
	sessions map[string]*Session
	tokens   map[string]string
	mu       sync.RWMutex
}

var (
	managerSingleton *Manager
	managerOnce      sync.Once
)

func GetManager() *Manager {
	managerOnce.Do(func() {
		managerSingleton = &Manager{
			sessions: make(map[string]*Session),
			tokens:   make(map[string]string),
		}
	})
	return managerSingleton
}

func (m *Manager) CreateSession() (string, error) {
	id, err := newSessionID()
	if err != nil {
		return "", err
	}

	token, err := newToken()
	if err != nil {
		return "", err
	}

	sessionDir := SessionDir(id)
	client, err := whatsapp.NewClient(context.Background(), sessionDir, m.makeEventHandler(id))
	if err != nil {
		return "", err
	}

	sess := &Session{ID: id, Token: token, Client: client}
	sess.UpdateStatusFromClient()

	if err := WriteToken(id, token); err != nil {
		return "", err
	}

	m.mu.Lock()
	m.sessions[id] = sess
	m.tokens[token] = id
	m.mu.Unlock()

	go m.Connect(sess)
	return token, nil
}

func (m *Manager) GetSession(id string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	sess, ok := m.sessions[id]
	return sess, ok
}

func (m *Manager) GetSessionByToken(token string) (*Session, bool) {
	m.mu.RLock()
	id, ok := m.tokens[token]
	m.mu.RUnlock()
	if !ok {
		return nil, false
	}
	return m.GetSession(id)
}

func (m *Manager) ListSessions() []SessionInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]SessionInfo, 0, len(m.sessions))
	for _, sess := range m.sessions {
		list = append(list, sess.Snapshot())
	}
	return list
}

func (m *Manager) RestoreSessionsOnStartup() error {
	ids, err := ListSessionIDs()
	if err != nil {
		return err
	}

	for _, id := range ids {
		sessionDir := SessionDir(id)
		client, err := whatsapp.NewClient(context.Background(), sessionDir, m.makeEventHandler(id))
		if err != nil {
			log.Printf("failed to restore session %s: %v", id, err)
			continue
		}

		token, err := ReadToken(id)
		if err != nil || token == "" {
			token, err = newToken()
			if err != nil {
				log.Printf("failed to create token for %s: %v", id, err)
				continue
			}
			if err := WriteToken(id, token); err != nil {
				log.Printf("failed to write token for %s: %v", id, err)
				continue
			}
		}

		sess := &Session{ID: id, Token: token, Client: client}
		sess.UpdateStatusFromClient()

		m.mu.Lock()
		m.sessions[id] = sess
		m.tokens[token] = id
		m.mu.Unlock()

		go m.Connect(sess)
	}

	return nil
}

func (m *Manager) Connect(session *Session) {
	if session == nil || session.Client == nil {
		return
	}

	session.Mutex.Lock()
	if session.Client.IsConnected() {
		session.Connected = true
		session.Mutex.Unlock()
		return
	}
	session.Mutex.Unlock()

	ctx := context.Background()
	if session.Client.Store.ID == nil {
		qrChan, err := session.Client.GetQRChannel(ctx)
		if err != nil {
			log.Printf("failed to get QR channel for %s: %v", session.ID, err)
		} else {
			go func() {
				for evt := range qrChan {
					switch evt.Event {
					case "code":
						session.SetQR(evt.Code)
					case "success", "timeout":
						session.SetQR("")
					}
				}
			}()
		}
	} else {
		session.SetQR("")
	}

	if err := session.Client.Connect(); err != nil {
		log.Printf("failed to connect session %s: %v", session.ID, err)
	}

	session.UpdateStatusFromClient()
}

func (m *Manager) GetQR(sessionID string) (string, error) {
	sess, ok := m.GetSession(sessionID)
	if !ok {
		return "", errors.New("session not found")
	}

	qr := sess.GetQR()
	if qr == "" {
		return "", errors.New("qr not available")
	}

	return whatsapp.QRToBase64PNG(qr)
}

func (m *Manager) GetQRByToken(token string) (string, error) {
	sess, ok := m.GetSessionByToken(token)
	if !ok {
		return "", errors.New("session not found")
	}

	qr := sess.GetQR()
	if qr == "" {
		return "", errors.New("qr not available")
	}

	return whatsapp.QRToBase64PNG(qr)
}

func (m *Manager) SendTextByToken(ctx context.Context, token string, phone string, message string) error {
	sess, ok := m.GetSessionByToken(token)
	if !ok {
		return errors.New("session not found")
	}

	if sess.Client == nil {
		return errors.New("session client not initialized")
	}
	if !sess.Client.IsConnected() {
		return errors.New("session not connected")
	}
	if sess.Client.Store.ID == nil {
		return errors.New("session not logged in")
	}

	jid := types.NewJID(phone, "s.whatsapp.net")
	_, err := sess.Client.SendMessage(ctx, jid, &waProto.Message{
		Conversation: proto.String(message),
	})
	return err
}

func (m *Manager) makeEventHandler(id string) func(interface{}) {
	return func(evt interface{}) {
		sess, ok := m.GetSession(id)
		if !ok {
			return
		}

		switch e := evt.(type) {
		case *events.Message:
			msg := extractTextMessage(e)
			if msg != nil {
				sess.AddMessage(*msg)
			}
		case *events.Connected:
			sess.SetConnected(true)
			sess.SetLoggedIn(sess.Client.Store.ID != nil)
			if sess.Client.Store.ID != nil {
				sess.SetJID(sess.Client.Store.ID.String())
			}
		case *events.Disconnected:
			sess.SetConnected(false)
			go m.reconnectWithDelay(sess, 5*time.Second)
		case *events.LoggedOut:
			sess.SetConnected(false)
			sess.SetLoggedIn(false)
			sess.SetJID("")
			sess.SetQR("")
		}
	}
}

func extractTextMessage(evt *events.Message) *IncomingMessage {
	if evt == nil || evt.Message == nil {
		return nil
	}

	text := ""
	if conv := evt.Message.GetConversation(); conv != "" {
		text = conv
	} else if ext := evt.Message.GetExtendedTextMessage(); ext != nil {
		text = ext.GetText()
	}

	if text == "" {
		return nil
	}

	return &IncomingMessage{
		From:      evt.Info.Sender.User,
		Name:      evt.Info.PushName,
		Message:   text,
		Timestamp: evt.Info.Timestamp.Unix(),
	}
}

func (m *Manager) reconnectWithDelay(sess *Session, delay time.Duration) {
	if sess == nil || sess.Client == nil {
		return
	}
	if sess.Client.IsConnected() {
		return
	}
	time.Sleep(delay)
	m.Connect(sess)
}

func newSessionID() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func newToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
