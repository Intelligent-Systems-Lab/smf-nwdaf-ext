// Package nsmf contains the Release 18 Nsmf_EventExposure wire types that are
// incomplete in the free5GC OpenAPI module currently used by this SMF.
//
// Source: 3GPP TS 29.508 V18.11.0, TS29508_Nsmf_EventExposure.yaml.
package nsmf

import (
	"github.com/free5gc/openapi/models"
	"github.com/free5gc/smf/internal/compat/nupf"
)

type Event string

const EventUPFEvent Event = "UPF_EVENT"

type EventSubscription struct {
	Event       Event                   `json:"event"`
	NetworkArea *models.NetworkAreaInfo `json:"networkArea,omitempty"`
	UPFEvents   []nupf.Event            `json:"upfEvents,omitempty"`
}

type EventExposure struct {
	SUPI        string                                    `json:"supi,omitempty"`
	AnyUEInd    bool                                      `json:"anyUeInd,omitempty"`
	GroupID     string                                    `json:"groupId,omitempty"`
	PduSeID     int32                                     `json:"pduSeId,omitempty"`
	Dnn         string                                    `json:"dnn,omitempty"`
	Snssai      *models.Snssai                            `json:"snssai,omitempty"`
	NFID        string                                    `json:"nfId,omitempty"`
	SubID       string                                    `json:"subId,omitempty"`
	NotifID     string                                    `json:"notifId"`
	NotifURI    string                                    `json:"notifUri"`
	EventSubs   []EventSubscription                       `json:"eventSubs"`
	NotifMethod models.SmfEventExposureNotificationMethod `json:"notifMethod,omitempty"`
	RepPeriod   int32                                     `json:"repPeriod,omitempty"`
}
