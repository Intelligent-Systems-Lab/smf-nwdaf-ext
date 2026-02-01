/*
 * SMF Nsmf_EventExposure in-memory subscription store
 *
 * This file defines the thread-safe, process-local state used by the
 * Nsmf_EventExposure Create/Delete handlers in Task2 Patch 1.
 */

package context

import (
	"sync"
	"time"
)

// NsmfEventExposureSubscriptionState stores the validated subscription data that
// the SMF needs to keep for lifecycle management (create/delete) in memory.
type NsmfEventExposureSubscriptionState struct {
	SubId                   string
	Supi                    string
	NotifId                 string
	NotifUri                string
	RepPeriod               int32
	NotifMethod             string
	MeasurementTypes        []string
	GranularityOfMeasure    string
	BundledEventNotifyUri   string
	UpfEventType            string
	SmfEventType            string
	UeIpAddress             string
	SelectedUpfApiRoot      string
	UpfSubscriptionId       string
	UpfSubscriptionLocation string
	CreatedAt               time.Time
}

type nsmfEventExposureStore struct {
	mu   sync.RWMutex
	byID map[string]*NsmfEventExposureSubscriptionState
}

func newNsmfEventExposureStore() *nsmfEventExposureStore {
	return &nsmfEventExposureStore{
		byID: make(map[string]*NsmfEventExposureSubscriptionState),
	}
}

var nsmfEventExposureSubscriptions = newNsmfEventExposureStore()

// StoreNsmfEventExposureSubscription stores a validated subscription state.
func StoreNsmfEventExposureSubscription(state *NsmfEventExposureSubscriptionState) {
	nsmfEventExposureSubscriptions.mu.Lock()
	defer nsmfEventExposureSubscriptions.mu.Unlock()
	nsmfEventExposureSubscriptions.byID[state.SubId] = state
}

// GetNsmfEventExposureSubscription returns a stored subscription by ID.
func GetNsmfEventExposureSubscription(subId string) (*NsmfEventExposureSubscriptionState, bool) {
	nsmfEventExposureSubscriptions.mu.RLock()
	defer nsmfEventExposureSubscriptions.mu.RUnlock()
	state, ok := nsmfEventExposureSubscriptions.byID[subId]
	return state, ok
}

// DeleteNsmfEventExposureSubscription removes a stored subscription by ID.
func DeleteNsmfEventExposureSubscription(subId string) (*NsmfEventExposureSubscriptionState, bool) {
	nsmfEventExposureSubscriptions.mu.Lock()
	defer nsmfEventExposureSubscriptions.mu.Unlock()
	state, ok := nsmfEventExposureSubscriptions.byID[subId]
	if ok {
		delete(nsmfEventExposureSubscriptions.byID, subId)
	}
	return state, ok
}
