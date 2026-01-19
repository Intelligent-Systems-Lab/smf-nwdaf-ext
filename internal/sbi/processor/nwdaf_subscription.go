// File: Task1 NWDAF subscription processor for UE_COMMUNICATION.
// References TS 29.520 (Create/Notify/Delete) and TS 23.288 (UE Communication analytics).
package processor

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/free5gc/openapi/models"
	smf_context "github.com/free5gc/smf/internal/context"
	"github.com/free5gc/smf/internal/logger"
	"github.com/free5gc/smf/pkg/factory"
)

type NwdafSubscriptionRequest struct {
	Supi            string                       `json:"supi"`
	NwdafApiRoot    string                       `json:"nwdafApiRoot"`
	NotificationURI string                       `json:"notificationURI"`
	NotifCorrId     string                       `json:"notifCorrId"`
	EvtReq          *models.ReportingInformation `json:"evtReq,omitempty"`
	RepPeriod       *int32                       `json:"repPeriod,omitempty"`
	RetryTimes      *int                         `json:"retryTimes,omitempty"`
	RetryIntervalMs *int                         `json:"retryIntervalMs,omitempty"`
}

// HandleOAMCreateNwdafSubscription triggers CreateNWDAFEventsSubscription using OAM or config defaults.
func (p *Processor) HandleOAMCreateNwdafSubscription(c *gin.Context, req *NwdafSubscriptionRequest) {
	if req == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "empty request"})
		return
	}
	// OAM request drives subscription creation; config defaults apply when fields are omitted.
	if req.Supi == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "supi is required"})
		return
	}

	resolved := resolveNwdafDefaults(p.Config(), p.Context(), req)
	if resolved.apiRoot == "" || resolved.notificationURI == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "nwdafApiRoot and notificationURI are required"})
		return
	}

	// TODO(V1): Only UE_COMMUNICATION with single SUPI is supported to keep the first patch minimal.
	if req.EvtReq != nil && req.RepPeriod != nil && *req.RepPeriod > 0 {
		// Override repPeriod when provided explicitly by OAM.
		req.EvtReq.RepPeriod = *req.RepPeriod
		if req.EvtReq.NotifMethod == "" {
			req.EvtReq.NotifMethod = models.SmfEventExposureNotificationMethod_PERIODIC
		}
	}
	subscription := models.NnwdafEventsSubscription{
		EventSubscriptions: []models.NwdafEventsSubscriptionEventSubscription{
			{
				Event: models.NwdafEvent_UE_COMMUNICATION,
				TgtUe: &models.TargetUeInformation{
					Supis: []string{req.Supi},
				},
			},
		},
		NotificationURI: resolved.notificationURI,
		EvtReq:          req.EvtReq,
	}
	if resolved.notifCorrId != "" {
		// notifCorrId is optional; include only when explicitly provided.
		subscription.NotifCorrId = resolved.notifCorrId
	}
	if subscription.EvtReq == nil && resolved.repPeriod > 0 {
		// Apply config default reporting period when OAM does not specify evtReq.
		subscription.EvtReq = &models.ReportingInformation{
			NotifMethod: models.SmfEventExposureNotificationMethod_PERIODIC,
			RepPeriod:   resolved.repPeriod,
		}
	}

	ctx := context.Background()
	location, err := p.createNwdafSubscriptionWithRetry(
		ctx,
		req.Supi,
		resolved,
		&subscription,
	)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	subscriptionId, parseErr := extractNwdafSubscriptionId(location)
	if parseErr != nil {
		logFields := logrus.Fields{
			logger.FieldSupi: req.Supi,
			"http_status":    http.StatusBadGateway,
		}
		if resolved.notifCorrId != "" {
			logFields["notif_corr_id"] = resolved.notifCorrId
		}
		logger.SBILog.WithFields(logFields).
			Errorf("NWDAF create returned invalid Location header: %s (%v)", location, parseErr)
		c.JSON(http.StatusBadGateway, gin.H{"error": "invalid Location header"})
		return
	}

	state := &smf_context.NwdafSubscriptionState{
		Supi:            req.Supi,
		SubscriptionId:  subscriptionId,
		NotifCorrId:     resolved.notifCorrId,
		NotificationURI: resolved.notificationURI,
		NwdafApiRoot:    resolved.apiRoot,
	}
	p.Context().NwdafSubs.Put(state)

	logFields := logrus.Fields{
		logger.FieldSupi:  req.Supi,
		"subscription_id": subscriptionId,
		"http_status":     http.StatusCreated,
	}
	if resolved.notifCorrId != "" {
		logFields["notif_corr_id"] = resolved.notifCorrId
	}
	logger.SBILog.WithFields(logFields).Info("NWDAF subscription created")

	c.Header("Location", location)
	c.JSON(http.StatusCreated, state)
}

// HandleOAMDeleteNwdafSubscription triggers DeleteNWDAFEventsSubscription by subscriptionId.
func (p *Processor) HandleOAMDeleteNwdafSubscription(c *gin.Context, subscriptionId string) {
	if subscriptionId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subscriptionId is required"})
		return
	}

	state, ok := p.Context().NwdafSubs.GetBySubscriptionId(subscriptionId)
	if !ok {
		logger.SBILog.WithFields(logrus.Fields{
			"subscription_id": subscriptionId,
			"http_status":     http.StatusNotFound,
		}).Warn("NWDAF subscription not found in local state")
		c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		return
	}

	ctx := context.Background()
	resolved := resolveNwdafDefaults(p.Config(), p.Context(), &NwdafSubscriptionRequest{})
	if err := p.deleteNwdafSubscriptionWithRetry(
		ctx,
		state,
		subscriptionId,
		resolved,
	); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	p.Context().NwdafSubs.DeleteBySubscriptionId(subscriptionId)

	logFields := logrus.Fields{
		logger.FieldSupi:  state.Supi,
		"subscription_id": subscriptionId,
		"http_status":     http.StatusNoContent,
	}
	if state.NotifCorrId != "" {
		logFields["notif_corr_id"] = state.NotifCorrId
	}
	logger.SBILog.WithFields(logFields).Info("NWDAF subscription deleted")

	c.Status(http.StatusNoContent)
}

// HandleOAMGetNwdafSubscription returns local subscription state for OAM debugging.
func (p *Processor) HandleOAMGetNwdafSubscription(c *gin.Context, subscriptionId string) {
	if subscriptionId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subscriptionId is required"})
		return
	}

	state, ok := p.Context().NwdafSubs.GetBySubscriptionId(subscriptionId)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		return
	}

	c.JSON(http.StatusOK, state)
}

// HandleNwdafNotification handles NWDAF notification array and responds with 204 on success.
func (p *Processor) HandleNwdafNotification(
	c *gin.Context,
	notifications []models.NnwdafEventsSubscriptionNotification,
) {
	for _, notif := range notifications {
		if strings.TrimSpace(notif.SubscriptionId) == "" {
			logger.SBILog.WithField("http_status", http.StatusBadRequest).
				Warn("NWDAF notification missing subscriptionId")
			c.Status(http.StatusBadRequest)
			return
		}

		// Task1 correlates notifications by subscriptionId; notifCorrId is optional.
		supi := ""
		if state, ok := p.Context().NwdafSubs.GetBySubscriptionId(notif.SubscriptionId); ok {
			supi = state.Supi
		}
		logFields := logrus.Fields{
			logger.FieldSupi:  supi,
			"subscription_id": notif.SubscriptionId,
			"http_status":     http.StatusNoContent,
		}
		if notif.NotifCorrId != "" {
			logFields["notif_corr_id"] = notif.NotifCorrId
		}
		logger.SBILog.WithFields(logFields).Info("NWDAF notification received")
	}

	c.Status(http.StatusNoContent)
}

var nwdafLocationRegexp = regexp.MustCompile(`/subscriptions/([^/]+)$`)

// extractNwdafSubscriptionId parses subscriptionId from the Location header.
func extractNwdafSubscriptionId(location string) (string, error) {
	// Parse subscriptionId from Location header (URL path preferred, regex as fallback).
	if strings.TrimSpace(location) == "" {
		return "", fmt.Errorf("empty Location header")
	}

	if parsed, err := url.Parse(location); err == nil && parsed.Path != "" {
		path := strings.TrimRight(parsed.Path, "/")
		parts := strings.Split(path, "/")
		if len(parts) > 0 && parts[len(parts)-1] != "" {
			return parts[len(parts)-1], nil
		}
	}

	match := nwdafLocationRegexp.FindStringSubmatch(location)
	if len(match) > 1 {
		return match[1], nil
	}
	return "", fmt.Errorf("cannot parse subscriptionId from Location")
}

// nwdafResolvedParams holds resolved runtime values for a subscription request.
type nwdafResolvedParams struct {
	apiRoot         string
	notificationURI string
	notifCorrId     string
	repPeriod       int32
	retryTimes      int
	retryInterval   time.Duration
}

// resolveNwdafDefaults merges OAM request values with smfcfg defaults and derived fallbacks.
func resolveNwdafDefaults(
	cfg *factory.Config,
	smfCtx *smf_context.SMFContext,
	req *NwdafSubscriptionRequest,
) nwdafResolvedParams {
	// Resolve values in order: OAM request > config defaults > derived fallback.
	resolved := nwdafResolvedParams{}
	if req != nil {
		resolved.apiRoot = strings.TrimSpace(req.NwdafApiRoot)
		resolved.notificationURI = strings.TrimSpace(req.NotificationURI)
		// notifCorrId is optional; only propagate when explicitly provided.
		resolved.notifCorrId = strings.TrimSpace(req.NotifCorrId)
		if req.RepPeriod != nil && *req.RepPeriod > 0 {
			resolved.repPeriod = *req.RepPeriod
		}
		if req.RetryTimes != nil && *req.RetryTimes > 0 {
			resolved.retryTimes = *req.RetryTimes
		}
		if req.RetryIntervalMs != nil && *req.RetryIntervalMs > 0 {
			resolved.retryInterval = time.Duration(*req.RetryIntervalMs) * time.Millisecond
		}
	}

	if cfg != nil && cfg.Configuration != nil && cfg.Configuration.NwdafSubscription != nil {
		conf := cfg.Configuration.NwdafSubscription
		if resolved.apiRoot == "" {
			resolved.apiRoot = strings.TrimSpace(conf.DefaultNwdafApiRoot)
		}
		if resolved.notificationURI == "" {
			resolved.notificationURI = strings.TrimSpace(conf.DefaultNotificationURI)
		}
		if resolved.notifCorrId == "" {
			resolved.notifCorrId = strings.TrimSpace(conf.DefaultNotifCorrId)
		}
		if resolved.repPeriod == 0 && conf.DefaultRepPeriod > 0 {
			resolved.repPeriod = conf.DefaultRepPeriod
		}
		if resolved.retryTimes == 0 && conf.RetryTimes > 0 {
			resolved.retryTimes = conf.RetryTimes
		}
		if resolved.retryInterval == 0 && conf.RetryInterval > 0 {
			resolved.retryInterval = conf.RetryInterval
		}
	}

	if resolved.notificationURI == "" && smfCtx != nil {
		// Derive callback URI from SMF SBI binding as a last resort.
		resolved.notificationURI = fmt.Sprintf(
			"%s://%s:%d%s",
			smfCtx.URIScheme,
			smfCtx.RegisterIPv4,
			smfCtx.SBIPort,
			factory.NwdafCallbackUriPrefix,
		)
	}
	if resolved.notifCorrId == "" {
		// notifCorrId is optional; leave empty unless provided by OAM or config.
	}
	return resolved
}

// createNwdafSubscriptionWithRetry wraps CreateNWDAFEventsSubscription with minimal retry/backoff.
func (p *Processor) createNwdafSubscriptionWithRetry(
	ctx context.Context,
	supi string,
	resolved nwdafResolvedParams,
	subscription *models.NnwdafEventsSubscription,
) (string, error) {
	// Minimal retry/backoff to improve robustness when NWDAF is transiently unavailable.
	attempts := resolved.retryTimes + 1
	if attempts < 1 {
		attempts = 1
	}
	var lastErr error
	var location string
	for attempt := 1; attempt <= attempts; attempt++ {
		location, _, lastErr = p.Consumer().SendCreateNwdafEventsSubscription(ctx, resolved.apiRoot, subscription)
		if lastErr == nil {
			return location, nil
		}
		logFields := logrus.Fields{
			logger.FieldSupi: supi,
			"attempt":        attempt,
			"http_status":    http.StatusBadGateway,
		}
		if resolved.notifCorrId != "" {
			logFields["notif_corr_id"] = resolved.notifCorrId
		}
		logger.SBILog.WithFields(logFields).Warnf("NWDAF create subscription failed: %v", lastErr)
		if attempt < attempts && resolved.retryInterval > 0 {
			time.Sleep(resolved.retryInterval)
		}
	}
	return "", lastErr
}

// deleteNwdafSubscriptionWithRetry wraps DeleteNWDAFEventsSubscription with minimal retry/backoff.
func (p *Processor) deleteNwdafSubscriptionWithRetry(
	ctx context.Context,
	state *smf_context.NwdafSubscriptionState,
	subscriptionId string,
	resolved nwdafResolvedParams,
) error {
	// Mirror create retry/backoff behavior for delete.
	attempts := resolved.retryTimes + 1
	if attempts < 1 {
		attempts = 1
	}
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		lastErr = p.Consumer().SendDeleteNwdafEventsSubscription(ctx, state.NwdafApiRoot, subscriptionId)
		if lastErr == nil {
			return nil
		}
		logFields := logrus.Fields{
			logger.FieldSupi:  state.Supi,
			"subscription_id": subscriptionId,
			"attempt":         attempt,
			"http_status":     http.StatusBadGateway,
		}
		if state.NotifCorrId != "" {
			logFields["notif_corr_id"] = state.NotifCorrId
		}
		logger.SBILog.WithFields(logFields).Warnf("NWDAF delete subscription failed: %v", lastErr)
		if attempt < attempts && resolved.retryInterval > 0 {
			time.Sleep(resolved.retryInterval)
		}
	}
	return lastErr
}
