// File: NWDAF callback entry point for UE_COMMUNICATION subscriptions.
// References TS 29.520 (Nnwdaf_EventsSubscription) and TS 23.288 (UE Communication analytics).
package sbi

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/models"
	"github.com/free5gc/smf/internal/logger"
)

// getNwdafCallbackRoutes registers the callback endpoint for NWDAF notifications.
func (s *Server) getNwdafCallbackRoutes() []Route {
	return []Route{
		{
			Name:    "NwdafEventsNotification",
			Method:  http.MethodPost,
			Pattern: "/",
			APIFunc: s.HTTPNwdafEventsNotification,
		},
	}
}

// HTTPNwdafEventsNotification handles NWDAF notifications for UE_COMMUNICATION and returns 204 on success.
func (s *Server) HTTPNwdafEventsNotification(c *gin.Context) {
	var notifications []models.NnwdafEventsSubscriptionNotification

	// Callback handler expects a JSON array per TS 29.520 contract.
	reqBody, err := c.GetRawData()
	if err != nil {
		logger.SBILog.WithField("http_status", http.StatusBadRequest).
			Errorln("NWDAF callback GetRawData failed")
		c.Status(http.StatusBadRequest)
		return
	}

	if err = openapi.Deserialize(&notifications, reqBody, c.ContentType()); err != nil {
		logger.SBILog.WithField("http_status", http.StatusBadRequest).
			Errorf("NWDAF callback deserialize failed: %v", err)
		c.Status(http.StatusBadRequest)
		return
	}
	if len(notifications) == 0 {
		logger.SBILog.WithField("http_status", http.StatusBadRequest).
			Warn("NWDAF callback received empty notifications array")
		c.Status(http.StatusBadRequest)
		return
	}

	s.Processor().HandleNwdafNotification(c, notifications)
}
