package telegram_bot

import (
	"sync"
	"time"
)

// UserSession хранит состояние пользователя
type UserSession struct {
	State     string
	TempName  string
	UpdatedAt time.Time
}

// SessionManager управляет всеми сессиями пользователей
type SessionManager struct {
	sessions map[int64]*UserSession
	mu       sync.RWMutex
}

// NewSessionManager создаёт новый менеджер сессий
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[int64]*UserSession),
	}
}

// getOrCreateSession возвращает существующую сессию или создаёт новую
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

// setState обновляет состояние пользователя
func (sm *SessionManager) setState(userID int64, state string) {
	session, _ := sm.getOrCreateSession(userID)
	session.State = state
}

// getState возвращает состояние пользователя
func (sm *SessionManager) getState(userID int64) (string, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[userID]
	if !exists {
		return "", false
	}
	return session.State, true
}

// setTempName сохраняет временное имя портфеля
func (sm *SessionManager) setTempName(userID int64, tempName string) {
	session, _ := sm.getOrCreateSession(userID)
	session.TempName = tempName
}

// getTempName возвращает временное имя портфеля
func (sm *SessionManager) getTempName(userID int64) (string, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[userID]
	if !exists {
		return "", false
	}
	return session.TempName, true
}

// clearSession удаляет сессию пользователя
func (sm *SessionManager) clearSession(userID int64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.sessions, userID)
}

// cleanOldSessions удаляет сессии, которые не обновлялись дольше заданного времени
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
