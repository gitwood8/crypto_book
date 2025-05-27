package telegram_bot

import (
	"sync"
	"time"

	"gitlab.com/avolkov/wood_post/pkg/log"
)

// setState updates the user's session state and refreshes session timestamp.

// save user's state
type UserSession struct {
	State                 string
	TempPortfolioName     string
	SelectedPortfolioName string
	BotMessageID          int
	UpdatedAt             time.Time
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
func (sm *SessionManager) getOrCreateSession(tgUserID int64) (*UserSession, bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[tgUserID]
	if !exists {
		session = &UserSession{}
		sm.sessions[tgUserID] = session
	}
	session.UpdatedAt = time.Now()
	return session, exists
}

// update user state
func (sm *SessionManager) setState(tgUserID int64, state string) {
	session, _ := sm.getOrCreateSession(tgUserID)
	session.State = state
}

// return user state
func (sm *SessionManager) getState(tgUserID int64) (string, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[tgUserID]
	if !exists {
		return "", false
	}
	return session.State, true
}

func (sm *SessionManager) setTempField(tgUserID int64, field string, value interface{}) {
	session, _ := sm.getOrCreateSession(tgUserID)

	switch field {
	case "TempPortfolioName":
		if v, ok := value.(string); ok {
			session.TempPortfolioName = v
		}
	case "SelectedPortfolioName":
		if v, ok := value.(string); ok {
			session.SelectedPortfolioName = v
		}
	case "BotMessageID":
		if v, ok := value.(int); ok {
			session.BotMessageID = v
		}
	case "NextAction":
		if v, ok := value.(int); ok {
			session.BotMessageID = v
		}
	default:
		log.Errorf("nknown field name: %s", field)
	}
}

// return temporary portfolio name
// func (sm *SessionManager) getTempName(tgUserID int64) (string, bool) {
// 	sm.mu.RLock()
// 	defer sm.mu.RUnlock()

// 	session, exists := sm.sessions[tgUserID]
// 	if !exists {
// 		return "", false
// 	}
// 	return session.TempPortfolioName, true
// }

func (sm *SessionManager) getSessionVars(tgUserID int64) (*UserSession, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[tgUserID]
	return session, exists
}

// delete user session
func (sm *SessionManager) clearSession(tgUserID int64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.sessions, tgUserID)
}

// delete sessions, that were not updated more then defined period
func (sm *SessionManager) cleanOldSessions(timeout time.Duration) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	for tgUserID, session := range sm.sessions {
		if now.Sub(session.UpdatedAt) > timeout {
			delete(sm.sessions, tgUserID)
		}
	}
}
