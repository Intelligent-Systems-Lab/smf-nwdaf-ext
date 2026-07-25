package processor

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/free5gc/openapi/models"
	"github.com/free5gc/smf/internal/compat/nupf"
	smf_context "github.com/free5gc/smf/internal/context"
	"github.com/free5gc/smf/internal/logger"
)

const (
	eventExposureGatewayFailureDetail = "Unable to complete the downstream service request."
	maxEventExposureIDAttempts        = 3
)

type EventExposureCreateRequest struct {
	Supi             string
	Selectors        smf_context.EventExposureSelectors
	NFID             string
	NotifID          string
	NotifURI         string
	MeasurementTypes []nupf.MeasurementType
	ReportingPeriod  int32
}

type EventExposureCreateResult struct {
	Subscription smf_context.EventExposureSubscription
}

type EventExposureProblem struct {
	Status         int
	ProblemDetails models.ProblemDetails
}

func (p *Processor) CreateEventExposureSubscription(
	ctx context.Context,
	request EventExposureCreateRequest,
) (EventExposureCreateResult, *EventExposureProblem) {
	target, err := p.eventExposure.Resolver.ResolveEventExposureTarget(ctx, request.Supi, request.Selectors)
	if err != nil {
		return EventExposureCreateResult{}, problemFromResolverError(err)
	}

	nupfRequest := buildNupfCreateEventSubscriptionRequest(p.Context().NfInstanceID, target, request)
	nupfConsumer := p.nupfEventExposureConsumer()
	if nupfConsumer == nil {
		return EventExposureCreateResult{}, eventExposureInternalProblem()
	}
	nupfResult, err := nupfConsumer.CreateSubscription(ctx, target, nupfRequest)
	if err != nil {
		return EventExposureCreateResult{}, eventExposureBadGatewayProblem()
	}

	subscription := smf_context.EventExposureSubscription{
		Supi:               request.Supi,
		Selectors:          request.Selectors,
		NFID:               request.NFID,
		NotifID:            request.NotifID,
		NotifURI:           request.NotifURI,
		MeasurementTypes:   append([]nupf.MeasurementType(nil), request.MeasurementTypes...),
		ReportingPeriod:    request.ReportingPeriod,
		Granularity:        nupf.GranularityPerSession,
		CreatedAt:          time.Now(),
		Target:             target,
		NupfSubscriptionID: nupfResult.SubscriptionID,
		NupfLocation:       nupfResult.ValidatedLocation,
	}

	for attempt := 0; attempt < maxEventExposureIDAttempts; attempt++ {
		subscription.ID = p.eventExposure.UUIDGenerator.NewString()
		err = p.eventExposure.Repository.Store(subscription)
		if err == nil {
			stored, _ := p.eventExposure.Repository.Get(subscription.ID)
			return EventExposureCreateResult{Subscription: stored}, nil
		}
		if !errors.Is(err, smf_context.ErrEventExposureSubscriptionIDCollision) {
			logger.SBILog.Warn("Event Exposure downstream subscription may be orphaned after local store failure")
			return EventExposureCreateResult{}, eventExposureInternalProblem()
		}
	}

	logger.SBILog.Warn("Event Exposure downstream subscription may be orphaned after subscription ID collision")
	return EventExposureCreateResult{}, eventExposureInternalProblem()
}

func (p *Processor) DeleteEventExposureSubscription(ctx context.Context, subscriptionID string) *EventExposureProblem {
	subscription, ok := p.eventExposure.Repository.ClaimDelete(subscriptionID)
	if !ok {
		return eventExposureNotFoundProblem()
	}

	nupfConsumer := p.nupfEventExposureConsumer()
	if nupfConsumer == nil {
		logger.SBILog.Warn("Event Exposure downstream delete did not start; local cleanup completed")
		return nil
	}
	if err := nupfConsumer.DeleteSubscription(ctx, subscription.Target, subscription.NupfSubscriptionID); err != nil {
		logger.SBILog.Warn("Event Exposure downstream delete did not complete; local cleanup completed")
	}
	return nil
}

func (p *Processor) nupfEventExposureConsumer() NupfEventExposureConsumer {
	if p.eventExposure.NupfConsumer != nil {
		return p.eventExposure.NupfConsumer
	}
	return p.Consumer()
}

func buildNupfCreateEventSubscriptionRequest(
	nfID string,
	target smf_context.EventExposureTarget,
	request EventExposureCreateRequest,
) nupf.CreateEventSubscription {
	return nupf.CreateEventSubscription{
		Subscription: nupf.EventSubscription{
			EventList: []nupf.Event{
				{
					Type:                     nupf.EventTypeUserDataUsageMeasures,
					MeasurementTypes:         append([]nupf.MeasurementType(nil), request.MeasurementTypes...),
					GranularityOfMeasurement: nupf.GranularityPerSession,
				},
			},
			EventNotifyURI:      request.NotifURI,
			NotifyCorrelationID: request.NotifID,
			EventReportingMode: nupf.EventMode{
				Trigger:   nupf.EventTriggerPeriodic,
				RepPeriod: request.ReportingPeriod,
			},
			NFID:        nfID,
			UEIPAddress: ipAddrFromNetIP(target.UEIPAddress),
		},
	}
}

func ipAddrFromNetIP(ip net.IP) *models.IpAddr {
	if ip == nil {
		return nil
	}
	if ipv4 := ip.To4(); ipv4 != nil {
		return &models.IpAddr{Ipv4Addr: ipv4.String()}
	}
	return &models.IpAddr{Ipv6Addr: ip.String()}
}

func problemFromResolverError(err error) *EventExposureProblem {
	switch {
	case errors.Is(err, smf_context.ErrEventExposureNoMatchingSession),
		errors.Is(err, smf_context.ErrEventExposureMissingUEIP),
		errors.Is(err, smf_context.ErrEventExposureMissingUPF):
		return eventExposureProblem(http.StatusNotFound, "Not Found", "Matching active session was not found.", "")
	case errors.Is(err, smf_context.ErrEventExposureMultipleSessions):
		return eventExposureProblem(http.StatusConflict, "Conflict", "Multiple matching active sessions were found.", "")
	case errors.Is(err, smf_context.ErrEventExposureMissingEndpoint):
		return eventExposureProblem(http.StatusServiceUnavailable, "Service Unavailable",
			"Selected UPF does not expose Event Exposure.", "")
	default:
		return eventExposureInternalProblem()
	}
}

func eventExposureBadGatewayProblem() *EventExposureProblem {
	return eventExposureProblem(http.StatusBadGateway, "Bad Gateway", eventExposureGatewayFailureDetail, "")
}

func eventExposureInternalProblem() *EventExposureProblem {
	return eventExposureProblem(http.StatusInternalServerError, "Internal Server Error",
		"Unable to persist the Event Exposure subscription.", "SYSTEM_FAILURE")
}

func eventExposureNotFoundProblem() *EventExposureProblem {
	return eventExposureProblem(http.StatusNotFound, "Not Found", "Event Exposure subscription was not found.", "")
}

func eventExposureProblem(status int, title, detail, cause string) *EventExposureProblem {
	return &EventExposureProblem{
		Status: status,
		ProblemDetails: models.ProblemDetails{
			Status: int32(status),
			Title:  title,
			Detail: detail,
			Cause:  cause,
		},
	}
}
