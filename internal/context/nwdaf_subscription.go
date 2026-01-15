package context

// NWDAF subscription store:
// - Tracks TS 29.520 subscription state (Location -> subscriptionId) for Create/Delete/Notify.

import (
	"fmt"
	"sync"
)

type NwdafSubKey struct {
	Supi           string
	SubscriptionId string
	NotifCorrId    string
}

type NwdafSubscriptionState struct {
	Supi            string `json:"supi"`
	SubscriptionId  string `json:"subscriptionId"`
	NotifCorrId     string `json:"notifCorrId"`
	NotificationURI string `json:"notificationURI"`
	NwdafApiRoot    string `json:"nwdafApiRoot"`
}

type NwdafSubStore struct {
	mu sync.RWMutex
	// Invariant: byKey is the source of truth keyed by (supi, subscriptionId, notifCorrId).
	byKey map[NwdafSubKey]*NwdafSubscriptionState
	// bySubscriptionId enables Delete/Get by subscriptionId when notifCorrId is unavailable.
	bySubscriptionId map[string]NwdafSubKey
	// bySubCorr enables callback lookup by (subscriptionId, notifCorrId); notifCorrId may be empty.
	bySubCorr map[string]NwdafSubKey
}

func NewNwdafSubStore() *NwdafSubStore {
	return &NwdafSubStore{
		byKey:            make(map[NwdafSubKey]*NwdafSubscriptionState),
		bySubscriptionId: make(map[string]NwdafSubKey),
		bySubCorr:        make(map[string]NwdafSubKey),
	}
}

func (s *NwdafSubStore) Put(state *NwdafSubscriptionState) {
	if state == nil {
		return
	}
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

func subCorrKey(subscriptionId, notifCorrId string) string {
	return fmt.Sprintf("%s:%s", subscriptionId, notifCorrId)
}
