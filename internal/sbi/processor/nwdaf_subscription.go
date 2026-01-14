package processor

import (
	"context"
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/free5gc/openapi/models"
	smf_context "github.com/free5gc/smf/internal/context"
	"github.com/free5gc/smf/internal/logger"
)

type NwdafSubscriptionRequest struct {
	Supi            string                       `json:"supi"`
	NwdafApiRoot    string                       `json:"nwdafApiRoot"`
	NotificationURI string                       `json:"notificationURI"`
	NotifCorrId     string                       `json:"notifCorrId"`
	EvtReq          *models.ReportingInformation `json:"evtReq,omitempty"`
}

func (p *Processor) HandleOAMCreateNwdafSubscription(c *gin.Context, req *NwdafSubscriptionRequest) {
	if req == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "empty request"})
		return
	}
	if req.Supi == "" || req.NwdafApiRoot == "" || req.NotificationURI == "" || req.NotifCorrId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "supi, nwdafApiRoot, notificationURI, notifCorrId are required"})
		return
	}

	// TODO(V1): Only UE_COMMUNICATION with single SUPI is supported for now.
	subscription := models.NnwdafEventsSubscription{
		EventSubscriptions: []models.EventSubscription{
			{
				Event: models.NwdafEvent_UE_COMMUNICATION,
				TgtUe: &models.TargetUeInfo{
					Supis: []string{req.Supi},
				},
			},
		},
		NotificationURI: req.NotificationURI,
		NotifCorrId:     req.NotifCorrId,
		EvtReq:          req.EvtReq,
	}

	ctx := context.Background()
	location, _, err := p.Consumer().SendCreateNwdafEventsSubscription(ctx, req.NwdafApiRoot, &subscription)
	if err != nil {
		logger.SBILog.WithFields(logrus.Fields{
			logger.FieldSupi: req.Supi,
			"notif_corr_id":  req.NotifCorrId,
			"http_status":    http.StatusBadGateway,
		}).Errorf("NWDAF create subscription failed: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	subscriptionId := extractNwdafSubscriptionId(location)
	if subscriptionId == "" {
		logger.SBILog.WithFields(logrus.Fields{
			logger.FieldSupi: req.Supi,
			"notif_corr_id":  req.NotifCorrId,
			"http_status":    http.StatusBadGateway,
		}).Errorf("NWDAF create returned invalid Location header: %s", location)
		c.JSON(http.StatusBadGateway, gin.H{"error": "invalid Location header"})
		return
	}

	state := &smf_context.NwdafSubscriptionState{
		Supi:            req.Supi,
		SubscriptionId:  subscriptionId,
		NotifCorrId:     req.NotifCorrId,
		NotificationURI: req.NotificationURI,
		NwdafApiRoot:    req.NwdafApiRoot,
	}
	p.Context().NwdafSubs.Put(state)

	logger.SBILog.WithFields(logrus.Fields{
		logger.FieldSupi:  req.Supi,
		"subscription_id": subscriptionId,
		"notif_corr_id":   req.NotifCorrId,
		"http_status":     http.StatusCreated,
	}).Info("NWDAF subscription created")

	c.Header("Location", location)
	c.JSON(http.StatusCreated, state)
}

func (p *Processor) HandleOAMDeleteNwdafSubscription(c *gin.Context, subscriptionId string) {
	if subscriptionId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subscriptionId is required"})
		return
	}

	state, ok := p.Context().NwdafSubs.GetBySubscriptionId(subscriptionId)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		return
	}

	ctx := context.Background()
	if err := p.Consumer().SendDeleteNwdafEventsSubscription(ctx, state.NwdafApiRoot, subscriptionId); err != nil {
		logger.SBILog.WithFields(logrus.Fields{
			logger.FieldSupi:  state.Supi,
			"subscription_id": subscriptionId,
			"notif_corr_id":   state.NotifCorrId,
			"http_status":     http.StatusBadGateway,
		}).Errorf("NWDAF delete subscription failed: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	p.Context().NwdafSubs.DeleteBySubscriptionId(subscriptionId)

	logger.SBILog.WithFields(logrus.Fields{
		logger.FieldSupi:  state.Supi,
		"subscription_id": subscriptionId,
		"notif_corr_id":   state.NotifCorrId,
		"http_status":     http.StatusNoContent,
	}).Info("NWDAF subscription deleted")

	c.Status(http.StatusNoContent)
}

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

func (p *Processor) HandleNwdafNotification(
	c *gin.Context,
	notifications []models.NnwdafEventsSubscriptionNotification,
) {
	for _, notif := range notifications {
		supi := ""
		if state, ok := p.Context().NwdafSubs.GetBySubCorr(notif.SubscriptionId, notif.NotifCorrId); ok {
			supi = state.Supi
		}

		logger.SBILog.WithFields(logrus.Fields{
			logger.FieldSupi:  supi,
			"subscription_id": notif.SubscriptionId,
			"notif_corr_id":   notif.NotifCorrId,
			"http_status":     http.StatusNoContent,
		}).Info("NWDAF notification received")
	}

	c.Status(http.StatusNoContent)
}

var nwdafLocationRegexp = regexp.MustCompile(`/subscriptions/([^/]+)$`)

func extractNwdafSubscriptionId(location string) string {
	match := nwdafLocationRegexp.FindStringSubmatch(location)
	if len(match) > 1 {
		return match[1]
	}
	return ""
}
