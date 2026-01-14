package sbi

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/models"
	"github.com/free5gc/smf/internal/logger"
)

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

func (s *Server) HTTPNwdafEventsNotification(c *gin.Context) {
	var notifications []models.NnwdafEventsSubscriptionNotification

	reqBody, err := c.GetRawData()
	if err != nil {
		logger.SBILog.Errorln("NWDAF callback GetRawData failed")
		c.Status(http.StatusBadRequest)
		return
	}

	if err = openapi.Deserialize(&notifications, reqBody, c.ContentType()); err != nil {
		logger.SBILog.Errorf("NWDAF callback deserialize failed: %v", err)
		c.Status(http.StatusBadRequest)
		return
	}

	s.Processor().HandleNwdafNotification(c, notifications)
}
