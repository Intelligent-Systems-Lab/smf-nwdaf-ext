package processor

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/free5gc/openapi/models"
	smf_context "github.com/free5gc/smf/internal/context"
	"github.com/free5gc/smf/internal/sbi/consumer"
	"github.com/free5gc/smf/pkg/factory"
)

type testProcessorApp struct {
	ctx      *smf_context.SMFContext
	cfg      *factory.Config
	consumer *consumer.Consumer
}

func (a *testProcessorApp) SetLogEnable(bool)    {}
func (a *testProcessorApp) SetLogLevel(string)   {}
func (a *testProcessorApp) SetReportCaller(bool) {}
func (a *testProcessorApp) Start()               {}
func (a *testProcessorApp) Terminate()           {}
func (a *testProcessorApp) Context() *smf_context.SMFContext {
	return a.ctx
}
func (a *testProcessorApp) Config() *factory.Config {
	return a.cfg
}
func (a *testProcessorApp) Consumer() *consumer.Consumer {
	return a.consumer
}

// TestHandleOAMCreateNwdafSubscription_BuildsContractPayload verifies the created NWDAF request matches contract fields.
func TestHandleOAMCreateNwdafSubscription_BuildsContractPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var received models.NnwdafEventsSubscription
	nwdafServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The contract requires POST /nnwdaf-eventssubscription/v1/subscriptions.
		require.Equal(t, "/nnwdaf-eventssubscription/v1/subscriptions", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)

		err := json.NewDecoder(r.Body).Decode(&received)
		require.NoError(t, err)

		w.Header().Set("Location", nwdafServer.URL+"/nnwdaf-eventssubscription/v1/subscriptions/sub123")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		require.NoError(t, json.NewEncoder(w).Encode(received))
	}))
	defer nwdafServer.Close()

	smfCtx := &smf_context.SMFContext{NwdafSubs: smf_context.NewNwdafSubStore()}
	cfg := &factory.Config{Configuration: &factory.Configuration{}}
	app := &testProcessorApp{ctx: smfCtx, cfg: cfg}
	smfConsumer, err := consumer.NewConsumer(app)
	require.NoError(t, err)
	app.consumer = smfConsumer
	p, err := NewProcessor(app)
	require.NoError(t, err)

	body := `{
		"supi":"imsi-208930000000001",
		"nwdafApiRoot":"` + nwdafServer.URL + `",
		"notificationURI":"http://127.0.0.2:8000/nwdaf-callback",
		"notifCorrId":"my-correlation-001",
		"evtReq":{"notifMethod":"PERIODIC","repPeriod":60}
	}`

	req := httptest.NewRequest(http.MethodPost, "/nsmf-oam/v1/nwdaf-subscriptions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req

	// Execute the OAM handler to build and send the NWDAF subscription request.
	p.HandleOAMCreateNwdafSubscription(c, &NwdafSubscriptionRequest{
		Supi:            "imsi-208930000000001",
		NwdafApiRoot:    nwdafServer.URL,
		NotificationURI: "http://127.0.0.2:8000/nwdaf-callback",
		NotifCorrId:     "my-correlation-001",
		EvtReq: &models.ReportingInformation{
			NotifMethod: models.SmfEventExposureNotificationMethod_PERIODIC,
			RepPeriod:   60,
		},
	})

	require.Equal(t, http.StatusCreated, rec.Code)
	require.Len(t, received.EventSubscriptions, 1)
	require.Equal(t, models.NwdafEvent_UE_COMMUNICATION, received.EventSubscriptions[0].Event)
	require.NotNil(t, received.EventSubscriptions[0].TgtUe)
	require.Equal(t, []string{"imsi-208930000000001"}, received.EventSubscriptions[0].TgtUe.Supis)
	require.Equal(t, "http://127.0.0.2:8000/nwdaf-callback", received.NotificationURI)
	require.Equal(t, "my-correlation-001", received.NotifCorrId)
	require.NotNil(t, received.EvtReq)
	require.Equal(t, int32(60), received.EvtReq.RepPeriod)
}

// TestExtractNwdafSubscriptionId validates Location header parsing robustness.
func TestExtractNwdafSubscriptionId(t *testing.T) {
	cases := []struct {
		name     string
		location string
		expect   string
	}{
		{
			name:     "full_url",
			location: "http://nwdaf.example/nnwdaf-eventssubscription/v1/subscriptions/sub123",
			expect:   "sub123",
		},
		{
			name:     "path_only",
			location: "/nnwdaf-eventssubscription/v1/subscriptions/sub456",
			expect:   "sub456",
		},
		{
			name:     "trailing_slash",
			location: "http://nwdaf.example/nnwdaf-eventssubscription/v1/subscriptions/sub789/",
			expect:   "sub789",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := extractNwdafSubscriptionId(tc.location)
			require.NoError(t, err)
			require.Equal(t, tc.expect, got)
		})
	}
}
