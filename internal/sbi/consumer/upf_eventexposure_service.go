package consumer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/free5gc/smf/internal/logger"
)

// Capabilities (V0):
// - Implemented: POST /ee-subscriptions, DELETE /ee-subscriptions/{subscriptionId}
// - Not implemented (stubbed by design): PATCH /ee-subscriptions/{subscriptionId}
type nupfEventExposureService struct {
	consumer *Consumer

	httpClientMu sync.RWMutex
	httpClient   *http.Client
}

// NupfEventExposureOperations enumerates TS 29.564 operations for future expansion.
type NupfEventExposureOperations interface {
	SendCreateNupfEventExposureSubscription(ctx context.Context, apiRoot string, payload any) (string, int, error)
	SendDeleteNupfEventExposureSubscription(
		ctx context.Context,
		apiRoot string,
		upfLocation string,
		subId string,
	) (int, string, error)
	// TODO: Implement PATCH /ee-subscriptions/{subscriptionId} when modify is needed.
}

func (s *nupfEventExposureService) getHTTPClient() *http.Client {
	s.httpClientMu.RLock()
	defer s.httpClientMu.RUnlock()
	if s.httpClient == nil {
		s.httpClient = &http.Client{}
	}
	return s.httpClient
}

// SendCreateNupfEventExposureSubscription sends CreateSubscription to UPF and returns Location header.
func (s *nupfEventExposureService) SendCreateNupfEventExposureSubscription(
	ctx context.Context,
	apiRoot string,
	payload any,
) (string, int, error) {
	target := normalizeNupfBasePath(apiRoot) + "/ee-subscriptions"
	body, err := json.Marshal(payload)
	if err != nil {
		return "", http.StatusInternalServerError, fmt.Errorf("marshal UPF request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		return "", http.StatusInternalServerError, fmt.Errorf("create UPF request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := s.getHTTPClient().Do(req)
	if err != nil {
		return "", http.StatusBadGateway, fmt.Errorf("call UPF create subscription: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			logger.ConsumerLog.Warnf("Failed to close UPF response body: %v", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusCreated {
		msg := ""
		if resp.Body != nil {
			if bodyBytes, readErr := io.ReadAll(resp.Body); readErr == nil {
				msg = strings.TrimSpace(string(bodyBytes))
			}
		}
		return "", resp.StatusCode, fmt.Errorf("UPF create failed: status=%d body=%s", resp.StatusCode, msg)
	}

	location := resp.Header.Get("Location")
	if location == "" {
		return "", resp.StatusCode, fmt.Errorf("UPF create missing Location header")
	}

	return location, resp.StatusCode, nil
}

// SendDeleteNupfEventExposureSubscription sends DeleteSubscription to UPF and returns status + body.
func (s *nupfEventExposureService) SendDeleteNupfEventExposureSubscription(
	ctx context.Context,
	apiRoot string,
	upfLocation string,
	subId string,
) (int, string, error) {
	target := resolveNupfDeleteTarget(apiRoot, upfLocation, subId)
	if target == "" {
		return http.StatusInternalServerError, "", fmt.Errorf("UPF delete target not resolved")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, target, nil)
	if err != nil {
		return http.StatusInternalServerError, "", fmt.Errorf("create UPF delete request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := s.getHTTPClient().Do(req)
	if err != nil {
		return http.StatusBadGateway, "", fmt.Errorf("call UPF delete subscription: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			logger.ConsumerLog.Warnf("Failed to close UPF response body: %v", closeErr)
		}
	}()

	body := ""
	if resp.Body != nil {
		if bodyBytes, readErr := io.ReadAll(resp.Body); readErr == nil {
			body = strings.TrimSpace(string(bodyBytes))
		}
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return resp.StatusCode, body, fmt.Errorf("UPF delete failed: status=%d", resp.StatusCode)
	}

	return resp.StatusCode, body, nil
}

func normalizeNupfBasePath(apiRoot string) string {
	base := strings.TrimRight(apiRoot, "/")
	if strings.Contains(base, "/nupf-ee/") {
		return base
	}
	return base + "/nupf-ee/v1"
}

func resolveNupfDeleteTarget(apiRoot string, upfLocation string, subId string) string {
	trimmedLocation := strings.TrimSpace(upfLocation)
	if trimmedLocation != "" {
		if strings.HasPrefix(trimmedLocation, "http://") || strings.HasPrefix(trimmedLocation, "https://") {
			return trimmedLocation
		}
		if apiRoot != "" {
			if baseURL, err := url.Parse(apiRoot); err == nil {
				baseURL.Path = ""
				baseURL.RawPath = ""
				baseURL.RawQuery = ""
				baseURL.Fragment = ""
				return strings.TrimRight(baseURL.String(), "/") + ensureLeadingSlash(trimmedLocation)
			}
		}
	}
	if apiRoot != "" && subId != "" {
		return normalizeNupfBasePath(apiRoot) + "/ee-subscriptions/" + subId
	}
	return ""
}

func ensureLeadingSlash(path string) string {
	if strings.HasPrefix(path, "/") {
		return path
	}
	return "/" + path
}
