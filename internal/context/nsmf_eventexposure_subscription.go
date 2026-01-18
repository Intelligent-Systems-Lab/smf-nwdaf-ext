package context

import "sync"

// NsmfEventExposureSubState stores the minimal subscription fields required for Task2 Patch1.
// Invariant: each subscription includes at least one UPF_EVENT with USER_DATA_USAGE_MEASURES.
type NsmfEventExposureSubState struct {
	NsmfSubId string
	Supi      string
	NotifId   string
	NotifUri  string
	EventSubs []NsmfEventExposureEventSubState
}

// NsmfEventExposureEventSubState mirrors the incoming event subscription for later UPF cascading.
type NsmfEventExposureEventSubState struct {
	Event     string
	UpfEvents []NsmfEventExposureUpfEventState
}

// NsmfEventExposureUpfEventState captures UPF event details needed for validation and Patch2.
type NsmfEventExposureUpfEventState struct {
	Type                     string
	MeasurementTypes         []string
	GranularityOfMeasurement string
}

// NsmfEventExposureSubStore tracks subscriptions by subId with concurrency protection.
type NsmfEventExposureSubStore struct {
	mu   sync.RWMutex
	byId map[string]*NsmfEventExposureSubState
}

// NewNsmfEventExposureSubStore creates an empty in-memory store.
func NewNsmfEventExposureSubStore() *NsmfEventExposureSubStore {
	return &NsmfEventExposureSubStore{
		byId: make(map[string]*NsmfEventExposureSubState),
	}
}

// Put inserts or overwrites a subscription by nsmfSubId.
func (s *NsmfEventExposureSubStore) Put(state *NsmfEventExposureSubState) {
	if state == nil || state.NsmfSubId == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byId[state.NsmfSubId] = state
}

// Get returns a subscription by nsmfSubId.
func (s *NsmfEventExposureSubStore) Get(subId string) (*NsmfEventExposureSubState, bool) {
	if subId == "" {
		return nil, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.byId[subId]
	return state, ok
}

// Delete removes and returns a subscription by nsmfSubId.
func (s *NsmfEventExposureSubStore) Delete(subId string) (*NsmfEventExposureSubState, bool) {
	if subId == "" {
		return nil, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.byId[subId]
	if ok {
		delete(s.byId, subId)
	}
	return state, ok
}
