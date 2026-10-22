package main

import (
	"log"
	"sync"
	"time"
)

// Session represents a user's chat session in a channel.
type Session struct {
	UserID     string
	ChannelID  string
	Messages   []DeepSeekMessage // Conversation history
	LastActive time.Time
	IsActive   bool
}

// SessionManager tracks all active sessions.
type SessionManager struct {
	sessions map[string]*Session // key: userID+channelID
	mu       sync.RWMutex
}

// NewSessionManager creates an empty SessionManager.
func NewSessionManager() *SessionManager {
	return &SessionManager{sessions: make(map[string]*Session)}
}

func getSessionKey(userID, channelID string) string {
	return userID + ":" + channelID
}

// StartSession starts (or refreshes) a session for a user in a channel.
func (sm *SessionManager) StartSession(userID, channelID string) *Session {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	key := getSessionKey(userID, channelID)
	if session, exists := sm.sessions[key]; exists {
		session.LastActive = time.Now()
		session.IsActive = true
		return session
	}

	session := &Session{
		UserID:     userID,
		ChannelID:  channelID,
		Messages:   make([]DeepSeekMessage, 0),
		LastActive: time.Now(),
		IsActive:   true,
	}
	sm.sessions[key] = session
	return session
}

// GetSession returns an active session, refreshing its last-active time.
func (sm *SessionManager) GetSession(userID, channelID string) (*Session, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	key := getSessionKey(userID, channelID)
	session, exists := sm.sessions[key]
	if !exists || !session.IsActive {
		return nil, false
	}
	session.LastActive = time.Now()
	return session, true
}

// EndSession deactivates a session and clears its history.
func (sm *SessionManager) EndSession(userID, channelID string) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	key := getSessionKey(userID, channelID)
	if session, exists := sm.sessions[key]; exists {
		session.IsActive = false
		session.Messages = make([]DeepSeekMessage, 0)
		return true
	}
	return false
}

// AddMessage appends a message, keeping only the most recent 20.
func (sm *SessionManager) AddMessage(userID, channelID, role, content string) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	key := getSessionKey(userID, channelID)
	session, exists := sm.sessions[key]
	if !exists || !session.IsActive {
		return false
	}

	session.Messages = append(session.Messages, DeepSeekMessage{Role: role, Content: content})
	session.LastActive = time.Now()

	if len(session.Messages) > 20 {
		session.Messages = session.Messages[len(session.Messages)-20:]
	}
	return true
}

// CleanupInactiveSessions expires sessions idle for longer than SessionTimeout.
func (sm *SessionManager) CleanupInactiveSessions() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	for _, session := range sm.sessions {
		if session.IsActive && now.Sub(session.LastActive) > SessionTimeout {
			session.IsActive = false
			session.Messages = make([]DeepSeekMessage, 0)
			log.Printf("Session timeout for user %s in channel %s", session.UserID, session.ChannelID)
		}
	}
}
