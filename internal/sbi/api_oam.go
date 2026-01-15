package sbi

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/free5gc/smf/internal/sbi/processor"
)

func (s *Server) getOAMRoutes() []Route {
	return []Route{
		{
			Name:    "Index",
			Method:  http.MethodGet,
			Pattern: "/",
			APIFunc: func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "Service Available"})
			},
		},
		{
			Name:    "Get UE PDU Session Info",
			Method:  http.MethodGet,
			Pattern: "/ue-pdu-session-info/:smContextRef",
			APIFunc: s.HTTPGetUEPDUSessionInfo,
		},
		{
			Name:    "Get SMF Userplane Information",
			Method:  http.MethodGet,
			Pattern: "/user-plane-info/",
			APIFunc: s.HTTPGetSMFUserPlaneInfo,
		},
		// NWDAF subscription OAM endpoints for manual trigger.
		{
			Name:    "Create NWDAF Subscription",
			Method:  http.MethodPost,
			Pattern: "/nwdaf-subscriptions",
			APIFunc: s.HTTPCreateNwdafSubscription,
		},
		// Delete by subscriptionId.
		{
			Name:    "Delete NWDAF Subscription",
			Method:  http.MethodDelete,
			Pattern: "/nwdaf-subscriptions/:subscriptionId",
			APIFunc: s.HTTPDeleteNwdafSubscription,
		},
		// Get by subscriptionId.
		{
			Name:    "Get NWDAF Subscription",
			Method:  http.MethodGet,
			Pattern: "/nwdaf-subscriptions/:subscriptionId",
			APIFunc: s.HTTPGetNwdafSubscription,
		},
	}
}

func (s *Server) HTTPGetUEPDUSessionInfo(c *gin.Context) {
	smContextRef := c.Params.ByName("smContextRef")

	s.Processor().HandleOAMGetUEPDUSessionInfo(c, smContextRef)
}

func (s *Server) HTTPGetSMFUserPlaneInfo(c *gin.Context) {
	s.Processor().HandleGetSMFUserPlaneInfo(c)
}

func (s *Server) HTTPCreateNwdafSubscription(c *gin.Context) {
	var req processor.NwdafSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	s.Processor().HandleOAMCreateNwdafSubscription(c, &req)
}

func (s *Server) HTTPDeleteNwdafSubscription(c *gin.Context) {
	subscriptionId := c.Params.ByName("subscriptionId")
	s.Processor().HandleOAMDeleteNwdafSubscription(c, subscriptionId)
}

func (s *Server) HTTPGetNwdafSubscription(c *gin.Context) {
	subscriptionId := c.Params.ByName("subscriptionId")
	s.Processor().HandleOAMGetNwdafSubscription(c, subscriptionId)
}
