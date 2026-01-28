package models

import (
	"sync"
)

// PortfolioManager handles concurrent portfolio updates safely
// Uses per-user locks instead of global lock
type PortfolioManager struct {
	userLocks map[int]*sync.Mutex // Map of user_id → mutex
	mapMutex  sync.RWMutex        // Protects the map itself
}

// NewPortfolioManager creates a new portfolio manager
func NewPortfolioManager() *PortfolioManager {
	return &PortfolioManager{
		userLocks: make(map[int]*sync.Mutex),
	}
}

// LockUser locks the portfolio for a specific user
func (pm *PortfolioManager) LockUser(userID int) {
	// First, get or create mutex for this user
	pm.mapMutex.Lock()

	if pm.userLocks[userID] == nil {
		pm.userLocks[userID] = &sync.Mutex{}
	}

	userMutex := pm.userLocks[userID]
	pm.mapMutex.Unlock()

	// Now lock that user's mutex
	userMutex.Lock()
}

// UnlockUser unlocks the portfolio for a specific user
func (pm *PortfolioManager) UnlockUser(userID int) {
	pm.mapMutex.RLock()
	userMutex := pm.userLocks[userID]
	pm.mapMutex.RUnlock()

	if userMutex != nil {
		userMutex.Unlock()
	}
}
