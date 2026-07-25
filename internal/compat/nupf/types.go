// Package nupf contains the Release 18 Nupf_EventExposure wire types that are
// absent from the free5GC OpenAPI module currently used by this SMF.
//
// Source: 3GPP TS 29.564 V18.8.0, TS29564_Nupf_EventExposure.yaml.
package nupf

import "github.com/free5gc/openapi/models"

type EventType string

const (
	EventTypeUserDataUsageMeasures EventType = "USER_DATA_USAGE_MEASURES"
)

type MeasurementType string

const (
	MeasurementTypeVolume     MeasurementType = "VOLUME_MEASUREMENT"
	MeasurementTypeThroughput MeasurementType = "THROUGHPUT_MEASUREMENT"
)

type GranularityOfMeasurement string

const (
	GranularityPerSession GranularityOfMeasurement = "PER_SESSION"
)

type EventTrigger string

const (
	EventTriggerPeriodic EventTrigger = "PERIODIC"
)

type Event struct {
	Type                     EventType                `json:"type"`
	MeasurementTypes         []MeasurementType        `json:"measurementTypes,omitempty"`
	GranularityOfMeasurement GranularityOfMeasurement `json:"granularityOfMeasurement,omitempty"`
}

type EventMode struct {
	Trigger   EventTrigger `json:"trigger"`
	RepPeriod int32        `json:"repPeriod,omitempty"`
}

type EventSubscription struct {
	EventList           []Event        `json:"eventList"`
	EventNotifyURI      string         `json:"eventNotifyUri"`
	NotifyCorrelationID string         `json:"notifyCorrelationId"`
	EventReportingMode  EventMode      `json:"eventReportingMode"`
	NFID                string         `json:"nfId"`
	UEIPAddress         *models.IpAddr `json:"ueIpAddress,omitempty"`
	SUPI                string         `json:"supi,omitempty"`
	DNN                 string         `json:"dnn,omitempty"`
	SNSSAI              *models.Snssai `json:"snssai,omitempty"`
}

type CreateEventSubscription struct {
	Subscription      EventSubscription `json:"subscription"`
	SupportedFeatures string            `json:"supportedFeatures,omitempty"`
}

type CreatedEventSubscription struct {
	Subscription      EventSubscription `json:"subscription"`
	SubscriptionID    string            `json:"subscriptionId"`
	SupportedFeatures string            `json:"supportedFeatures,omitempty"`
}
