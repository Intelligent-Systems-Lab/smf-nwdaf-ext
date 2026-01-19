package sbi

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	smf_context "github.com/free5gc/smf/internal/context"
	"github.com/free5gc/smf/internal/sbi/consumer"
	"github.com/free5gc/smf/internal/sbi/processor"
	"github.com/free5gc/smf/pkg/factory"
)

type testServerApp struct {
	ctx  *smf_context.SMFContext
	cfg  *factory.Config
	proc *processor.Processor
}

func (a *testServerApp) SetLogEnable(bool)    {}
func (a *testServerApp) SetLogLevel(string)   {}
func (a *testServerApp) SetReportCaller(bool) {}
func (a *testServerApp) Start()               {}
func (a *testServerApp) Terminate()           {}
func (a *testServerApp) Context() *smf_context.SMFContext {
	return a.ctx
}
func (a *testServerApp) Config() *factory.Config {
	return a.cfg
}
func (a *testServerApp) Consumer() *consumer.Consumer {
	return nil
}
func (a *testServerApp) Processor() *processor.Processor {
	return a.proc
}
func (a *testServerApp) CancelContext() context.Context {
	return context.Background()
}

// TestHTTPNwdafEventsNotification_OK verifies callback accepts a JSON array and returns 204.
func TestHTTPNwdafEventsNotification_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)

	smfCtx := &smf_context.SMFContext{NwdafSubs: smf_context.NewNwdafSubStore()}
	cfg := &factory.Config{Configuration: &factory.Configuration{}}
	app := &testServerApp{ctx: smfCtx, cfg: cfg}
	proc, err := processor.NewProcessor(app)
	require.NoError(t, err)
	app.proc = proc

	server := &Server{ServerSmf: app}

	body := `[
	  {
	    "subscriptionId": "sub-0001",
	    "notifCorrId": "my-correlation-001",
	    "eventNotifications": [
	      {
	        "event": "UE_COMMUNICATION",
	        "timeStampGen": "2026-01-14T09:00:00Z",
	        "ueComms": [
	          {
	            "ts": "2026-01-14T09:00:00Z",
	            "commDur": 3600,
	            "trafChar": {"ulVol": 1, "dlVol": 1}
	          }
	        ]
	      }
	    ]
	  }
	]`

	req := httptest.NewRequest(http.MethodPost, "/nwdaf-callback", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req

	// Exercise the callback handler with a valid array payload.
	server.HTTPNwdafEventsNotification(c)

	require.Equal(t, http.StatusNoContent, rec.Code)
}

// TestHTTPNwdafEventsNotification_Invalid verifies invalid payloads return 400.
func TestHTTPNwdafEventsNotification_Invalid(t *testing.T) {
	gin.SetMode(gin.TestMode)

	smfCtx := &smf_context.SMFContext{NwdafSubs: smf_context.NewNwdafSubStore()}
	cfg := &factory.Config{Configuration: &factory.Configuration{}}
	app := &testServerApp{ctx: smfCtx, cfg: cfg}
	proc, err := processor.NewProcessor(app)
	require.NoError(t, err)
	app.proc = proc

	server := &Server{ServerSmf: app}

	tests := []struct {
		name string
		body string
	}{
		{
			name: "empty_array",
			body: "[]",
		},
		{
			name: "not_array",
			body: `{"subscriptionId":"sub-0001"}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/nwdaf-callback", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = req

			// Exercise the callback handler with invalid payloads.
			server.HTTPNwdafEventsNotification(c)

			require.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}
