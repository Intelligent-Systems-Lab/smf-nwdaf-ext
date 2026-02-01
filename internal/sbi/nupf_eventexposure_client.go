/*
 * Nupf_EventExposure client helpers for Task2 Patch 2
 *
 * This file builds the UPF subscription request and performs the HTTP call to
 * the UPF Nupf_EventExposure API.
 */

package sbi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/free5gc/openapi/models"
	"github.com/free5gc/smf/pkg/factory"
)

type upfEventExposureCreateRequest struct {
	Subscription upfEventExposureSubscription `json:"subscription"`
}

// upfEventExposureSubscription matches TS 29.564 UpfEventSubscription shape.
// It is intentionally minimal and only carries fields required by Task2.
type upfEventExposureSubscription struct {
	NfId                string                     `json:"nfId"`
	UeIpAddress         string                     `json:"ueIpAddress"`
	EventList           []upfEventExposureEvent    `json:"eventList"`
	EventNotifyUri      string                     `json:"eventNotifyUri"`
	NotifyCorrelationId string                     `json:"notifyCorrelationId"`
	EventReportingMode  upfEventExposureReportMode `json:"eventReportingMode"`
}

type upfEventExposureEvent struct {
	Type                     string   `json:"type"`
	MeasurementTypes         []string `json:"measurementTypes,omitempty"`
	GranularityOfMeasurement string   `json:"granularityOfMeasurement,omitempty"`
}

// upfEventExposureReportMode maps the periodic reporting mode used by UPF.
type upfEventExposureReportMode struct {
	Trigger   string `json:"trigger"`
	RepPeriod int32  `json:"repPeriod,omitempty"`
}

func resolveUpfNfId() string {
	// Prefer explicit NUPF NF ID; fall back to SMF's nfInstanceId.
	cfg := factory.SmfConfig.Configuration
	if cfg.NupfEeNfId != "" {
		return cfg.NupfEeNfId
	}
	return cfg.NfInstanceId
}

func resolveUpfRequestTimeout() time.Duration {
	// Default timeout is small to avoid blocking the SBI handler for too long.
	timeout := factory.SmfConfig.Configuration.NupfEeReqTimeout
	if timeout == 0 {
		return 5 * time.Second
	}
	return timeout
}

func resolveUpfRetryCount() int {
	// Negative retry count is treated as zero for safety.
	retries := factory.SmfConfig.Configuration.NupfEeMaxRetries
	if retries < 0 {
		return 0
	}
	return retries
}

func createUpfEventExposureSubscription(
	ctx context.Context,
	apiRoot string,
	request upfEventExposureCreateRequest,
) (string, int, *models.ProblemDetails) {
	// Serialize the request once; retries reuse the same payload.
	payload, err := json.Marshal(request)
	if err != nil {
		return "", 0, problemDetailsSystemFailure("failed to marshal UPF subscription request")
	}

	// The UPF API path is fixed by TS 29.564.
	url := strings.TrimSuffix(apiRoot, "/") + "/nupf-ee/v1/ee-subscriptions"
	client := &http.Client{Timeout: resolveUpfRequestTimeout()}

	// Retry only on transport errors and 5xx responses.
	attempts := resolveUpfRetryCount() + 1
	for attempt := 0; attempt < attempts; attempt++ {
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
		if reqErr != nil {
			return "", 0, problemDetailsSystemFailure("failed to build UPF request")
		}
		req.Header.Set("Content-Type", "application/json")

		resp, doErr := client.Do(req)
		if doErr != nil {
			if attempt < attempts-1 {
				continue
			}
			return "", 0, problemDetailsBadGateway(fmt.Sprintf("UPF request failed: %v", doErr))
		}

		body, readErr := io.ReadAll(resp.Body)
		if closeErr := resp.Body.Close(); closeErr != nil {
			if readErr == nil {
				readErr = closeErr
			}
		}
		if readErr != nil {
			return "", resp.StatusCode, problemDetailsBadGateway("failed to read UPF response body")
		}

		if resp.StatusCode == http.StatusCreated {
			// Location is mandatory for Created; it is used as the UPF subscription identifier.
			location := resp.Header.Get("Location")
			if location == "" {
				return "", resp.StatusCode, problemDetailsBadGateway("UPF response missing Location header")
			}
			return location, resp.StatusCode, nil
		}

		if resp.StatusCode >= http.StatusInternalServerError && attempt < attempts-1 {
			continue
		}

		// Propagate UPF error details as a generic Bad Gateway for NWDAF.
		detail := fmt.Sprintf("UPF subscription failed with status %d", resp.StatusCode)
		if len(body) != 0 {
			detail = fmt.Sprintf("%s: %s", detail, strings.TrimSpace(string(body)))
		}
		return "", resp.StatusCode, problemDetailsBadGateway(detail)
	}

	return "", 0, problemDetailsBadGateway("UPF request failed after retries")
}

func problemDetailsBadGateway(detail string) *models.ProblemDetails {
	return &models.ProblemDetails{
		Title:  "Bad Gateway",
		Status: http.StatusBadGateway,
		Detail: detail,
		Cause:  "UPF_REQUEST_FAILED",
	}
}

func problemDetailsSystemFailure(detail string) *models.ProblemDetails {
	return &models.ProblemDetails{
		Title:  "System failure",
		Status: http.StatusInternalServerError,
		Detail: detail,
		Cause:  "SYSTEM_FAILURE",
	}
}

func deleteUpfEventExposureSubscription(
	ctx context.Context,
	apiRoot string,
	upfLocation string,
) (int, string, *models.ProblemDetails) {
	// If the UPF returned a full Location, use it verbatim; otherwise, build from apiRoot + ID.
	url := strings.TrimSpace(upfLocation)
	if url == "" {
		return 0, "", problemDetailsBadGateway("missing UPF subscription location")
	}
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = strings.TrimSuffix(apiRoot, "/") + "/nupf-ee/v1/ee-subscriptions/" + strings.TrimPrefix(url, "/")
	}

	client := &http.Client{Timeout: resolveUpfRequestTimeout()}
	attempts := resolveUpfRetryCount() + 1
	for attempt := 0; attempt < attempts; attempt++ {
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
		if reqErr != nil {
			return 0, "", problemDetailsSystemFailure("failed to build UPF delete request")
		}

		resp, doErr := client.Do(req)
		if doErr != nil {
			if attempt < attempts-1 {
				continue
			}
			return 0, "", problemDetailsBadGateway(fmt.Sprintf("UPF delete failed: %v", doErr))
		}

		body, readErr := io.ReadAll(resp.Body)
		if closeErr := resp.Body.Close(); closeErr != nil {
			if readErr == nil {
				readErr = closeErr
			}
		}
		if readErr != nil {
			return resp.StatusCode, "", problemDetailsBadGateway("failed to read UPF delete response body")
		}

		if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK {
			return resp.StatusCode, strings.TrimSpace(string(body)), nil
		}

		if resp.StatusCode >= http.StatusInternalServerError && attempt < attempts-1 {
			continue
		}

		detail := fmt.Sprintf("UPF delete failed with status %d", resp.StatusCode)
		if len(body) != 0 {
			detail = fmt.Sprintf("%s: %s", detail, strings.TrimSpace(string(body)))
		}
		return resp.StatusCode, strings.TrimSpace(string(body)), problemDetailsBadGateway(detail)
	}

	return 0, "", problemDetailsBadGateway("UPF delete failed after retries")
}
