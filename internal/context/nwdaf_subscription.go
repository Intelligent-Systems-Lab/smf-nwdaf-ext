// File: In-memory NWDAF subscription state for Task1 UE_COMMUNICATION flow.
// References TS 29.520 (Nnwdaf_EventsSubscription) and TS 23.288 (UE Communication analytics).
package context

import "sync"

// NwdafSubscriptionState stores the minimum state needed for callback and deletion.
type NwdafSubscriptionState struct {
	Supi            string `json:"supi"`
	SubscriptionId  string `json:"subscriptionId"`
	NotifCorrId     string `json:"notifCorrId"`
	NotificationURI string `json:"notificationURI"`
	NwdafApiRoot    string `json:"nwdafApiRoot"`
}

// NwdafSubStore indexes subscriptions by subscriptionId for callback and deletion.
type NwdafSubStore struct {
	mu sync.RWMutex
	// Invariant: bySubscriptionId is the source of truth keyed by subscriptionId.
	bySubscriptionId map[string]*NwdafSubscriptionState
}

// NewNwdafSubStore creates an empty subscription store.
func NewNwdafSubStore() *NwdafSubStore {
	return &NwdafSubStore{
		bySubscriptionId: make(map[string]*NwdafSubscriptionState),
	}
}

// Put inserts or updates state and maintains index invariants.
func (s *NwdafSubStore) Put(state *NwdafSubscriptionState) {
	if state == nil {
		return
	}
	// Store by subscriptionId because notifCorrId is optional in Task1.
	s.mu.Lock()
	defer s.mu.Unlock()

	s.bySubscriptionId[state.SubscriptionId] = state
}

// GetBySubscriptionId returns state by subscriptionId lookup.
func (s *NwdafSubStore) GetBySubscriptionId(subscriptionId string) (*NwdafSubscriptionState, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state, ok := s.bySubscriptionId[subscriptionId]
	return state, ok
}

// DeleteBySubscriptionId removes state and associated indexes.
func (s *NwdafSubStore) DeleteBySubscriptionId(subscriptionId string) (*NwdafSubscriptionState, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.bySubscriptionId[subscriptionId]
	if !ok {
		return nil, false
	}
	delete(s.bySubscriptionId, subscriptionId)
	return state, true
}
