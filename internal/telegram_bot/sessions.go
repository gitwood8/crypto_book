package telegram_bot

import (
	"sync"
	"time"
)

// save user's state
type UserSession struct {
	State     string
	TempName  string
	UpdatedAt time.Time
}

// manage all user's sessions
type SessionManager struct {
	sessions map[int64]*UserSession
	mu       sync.RWMutex
}

// create new session manager
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[int64]*UserSession),
	}
}

// return existing session or creates new
func (sm *SessionManager) getOrCreateSession(userID int64) (*UserSession, bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[userID]
	if !exists {
		session = &UserSession{}
		sm.sessions[userID] = session
	}
	session.UpdatedAt = time.Now()
	return session, exists
}

// update user state
func (sm *SessionManager) setState(userID int64, state string) {
	session, _ := sm.getOrCreateSession(userID)
	session.State = state
}

// return user state
func (sm *SessionManager) getState(userID int64) (string, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[userID]
	if !exists {
		return "", false
	}
	return session.State, true
}

// save temporary portfolio name
func (sm *SessionManager) setTempName(userID int64, tempName string) {
	session, _ := sm.getOrCreateSession(userID)
	session.TempName = tempName
}

// return temporary portfolio name
func (sm *SessionManager) getTempName(userID int64) (string, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[userID]
	if !exists {
		return "", false
	}
	return session.TempName, true
}

// delete user session
func (sm *SessionManager) clearSession(userID int64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.sessions, userID)
}

// delete sessions, that were not updated more then defined period
func (sm *SessionManager) cleanOldSessions(timeout time.Duration) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	for userID, session := range sm.sessions {
		if now.Sub(session.UpdatedAt) > timeout {
			delete(sm.sessions, userID)
		}
	}
}
