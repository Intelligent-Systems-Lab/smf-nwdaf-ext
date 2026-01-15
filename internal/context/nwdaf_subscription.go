// File: In-memory NWDAF subscription state for Task1 UE_COMMUNICATION flow.
// References TS 29.520 (Nnwdaf_EventsSubscription) and TS 23.288 (UE Communication analytics).
package context

import (
	"fmt"
	"sync"
)

// NwdafSubKey is the composite key used to track a single subscription instance.
type NwdafSubKey struct {
	Supi           string
	SubscriptionId string
	NotifCorrId    string
}

// NwdafSubscriptionState stores the minimum state needed for callback and deletion.
type NwdafSubscriptionState struct {
	Supi            string `json:"supi"`
	SubscriptionId  string `json:"subscriptionId"`
	NotifCorrId     string `json:"notifCorrId"`
	NotificationURI string `json:"notificationURI"`
	NwdafApiRoot    string `json:"nwdafApiRoot"`
}

// NwdafSubStore indexes subscriptions by composite key and common lookup fields.
type NwdafSubStore struct {
	mu sync.RWMutex
	// Invariant: byKey is the source of truth keyed by (supi, subscriptionId, notifCorrId).
	byKey map[NwdafSubKey]*NwdafSubscriptionState
	// bySubscriptionId enables Delete/Get by subscriptionId when notifCorrId is unavailable.
	bySubscriptionId map[string]NwdafSubKey
	// bySubCorr enables callback lookup by (subscriptionId, notifCorrId); notifCorrId may be empty.
	bySubCorr map[string]NwdafSubKey
}

// NewNwdafSubStore creates an empty subscription store.
func NewNwdafSubStore() *NwdafSubStore {
	return &NwdafSubStore{
		byKey:            make(map[NwdafSubKey]*NwdafSubscriptionState),
		bySubscriptionId: make(map[string]NwdafSubKey),
		bySubCorr:        make(map[string]NwdafSubKey),
	}
}

// Put inserts or updates state and maintains index invariants.
func (s *NwdafSubStore) Put(state *NwdafSubscriptionState) {
	if state == nil {
		return
	}
	// Keep indexes in sync to support lookup by (subscriptionId) or (subscriptionId+notifCorrId).
	key := NwdafSubKey{
		Supi:           state.Supi,
		SubscriptionId: state.SubscriptionId,
		NotifCorrId:    state.NotifCorrId,
	}
	subCorrKey := subCorrKey(state.SubscriptionId, state.NotifCorrId)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.byKey[key] = state
	s.bySubscriptionId[state.SubscriptionId] = key
	s.bySubCorr[subCorrKey] = key
}

// GetBySubscriptionId returns state by subscriptionId lookup.
func (s *NwdafSubStore) GetBySubscriptionId(subscriptionId string) (*NwdafSubscriptionState, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key, ok := s.bySubscriptionId[subscriptionId]
	if !ok {
		return nil, false
	}
	state, ok := s.byKey[key]
	return state, ok
}

// GetBySubCorr returns state by (subscriptionId, notifCorrId) lookup.
func (s *NwdafSubStore) GetBySubCorr(subscriptionId, notifCorrId string) (*NwdafSubscriptionState, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key, ok := s.bySubCorr[subCorrKey(subscriptionId, notifCorrId)]
	if !ok {
		return nil, false
	}
	state, ok := s.byKey[key]
	return state, ok
}

// DeleteBySubscriptionId removes state and associated indexes.
func (s *NwdafSubStore) DeleteBySubscriptionId(subscriptionId string) (*NwdafSubscriptionState, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key, ok := s.bySubscriptionId[subscriptionId]
	if !ok {
		return nil, false
	}
	state, ok := s.byKey[key]
	if !ok {
		delete(s.bySubscriptionId, subscriptionId)
		return nil, false
	}
	delete(s.byKey, key)
	delete(s.bySubscriptionId, subscriptionId)
	delete(s.bySubCorr, subCorrKey(subscriptionId, key.NotifCorrId))
	return state, true
}

// subCorrKey builds the compound key for notifCorrId lookup.
func subCorrKey(subscriptionId, notifCorrId string) string {
	return fmt.Sprintf("%s:%s", subscriptionId, notifCorrId)
}
