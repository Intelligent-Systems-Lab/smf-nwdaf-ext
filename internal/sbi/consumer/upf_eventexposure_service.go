package consumer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/free5gc/smf/internal/logger"
)

type nupfEventExposureService struct {
	consumer *Consumer

	httpClientMu sync.RWMutex
	httpClient   *http.Client
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

func normalizeNupfBasePath(apiRoot string) string {
	base := strings.TrimRight(apiRoot, "/")
	if strings.Contains(base, "/nupf-ee/") {
		return base
	}
	return base + "/nupf-ee/v1"
}
