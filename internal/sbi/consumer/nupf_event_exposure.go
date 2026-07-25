package consumer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/free5gc/openapi"
	"github.com/free5gc/openapi/models"
	"github.com/free5gc/smf/internal/compat/nupf"
	smf_context "github.com/free5gc/smf/internal/context"
	sbi_metrics "github.com/free5gc/util/metrics/sbi"
	"golang.org/x/oauth2"
)

const (
	nupfEventExposureTimeout   = 30 * time.Second
	nupfEventExposureBodyLimit = 1 << 20
	nupfServiceName            = models.ServiceName("nupf-ee")
)

type NupfEventExposureErrorKind string

const (
	NupfEventExposureErrorToken            NupfEventExposureErrorKind = "token"
	NupfEventExposureErrorRedirect         NupfEventExposureErrorKind = "redirect"
	NupfEventExposureErrorTransport        NupfEventExposureErrorKind = "transport"
	NupfEventExposureErrorUpstreamProblem  NupfEventExposureErrorKind = "upstream_problem"
	NupfEventExposureErrorMalformedSuccess NupfEventExposureErrorKind = "malformed_success"
)

type NupfEventExposureError struct {
	Kind           NupfEventExposureErrorKind
	StatusCode     int
	ProblemDetails *models.ProblemDetails
	Operation      string
}

func (e *NupfEventExposureError) Error() string {
	return fmt.Sprintf("nupf event exposure %s failed: %s", e.Operation, e.Kind)
}

type nupfEventExposureService struct {
	consumer *Consumer

	EventExposureMu             sync.RWMutex
	EventExposureClients        map[string]*http.Client
	EventExposureCreateRequests map[string]string
}

func (s *nupfEventExposureService) getEventExposureClient(apiRoot string) *http.Client {
	if apiRoot == "" {
		return nil
	}

	s.EventExposureMu.RLock()
	client, ok := s.EventExposureClients[apiRoot]
	s.EventExposureMu.RUnlock()
	if ok {
		return client
	}

	client = &http.Client{
		Timeout:       nupfEventExposureTimeout,
		CheckRedirect: rejectNupfRedirects,
	}
	requestURI := strings.TrimRight(apiRoot, "/") + "/nupf-ee/v1/ee-subscriptions"

	s.EventExposureMu.Lock()
	defer s.EventExposureMu.Unlock()
	if existing, exists := s.EventExposureClients[apiRoot]; exists {
		return existing
	}
	s.EventExposureClients[apiRoot] = client
	s.EventExposureCreateRequests[apiRoot] = requestURI
	return client
}

func (s *nupfEventExposureService) CreateSubscription(
	ctx context.Context,
	target smf_context.EventExposureTarget,
	request nupf.CreateEventSubscription,
) (smf_context.NupfCreateResult, error) {
	client := s.getEventExposureClient(target.APIroot)
	if client == nil {
		return smf_context.NupfCreateResult{}, nupfTransportError("create")
	}

	callCtx, err := s.tokenContext(ctx)
	if err != nil {
		return smf_context.NupfCreateResult{}, &NupfEventExposureError{
			Kind:      NupfEventExposureErrorToken,
			Operation: "create",
		}
	}

	payload, err := json.Marshal(request)
	if err != nil {
		return smf_context.NupfCreateResult{}, nupfTransportError("create")
	}
	requestURI := s.createRequestURI(target.APIroot)
	httpRequest, err := http.NewRequestWithContext(
		callCtx, http.MethodPost, requestURI, bytes.NewReader(payload))
	if err != nil {
		return smf_context.NupfCreateResult{}, nupfTransportError("create")
	}
	httpRequest.Header.Set("Accept", "application/json")
	httpRequest.Header.Set("Content-Type", "application/json")
	if err = authorizeNupfRequest(callCtx, httpRequest); err != nil {
		return smf_context.NupfCreateResult{}, &NupfEventExposureError{
			Kind:      NupfEventExposureErrorToken,
			Operation: "create",
		}
	}

	response, body, err := doNupfRequest(client, httpRequest)
	if err != nil {
		return smf_context.NupfCreateResult{}, classifyNupfHTTPError("create", response, body, err)
	}
	if response.StatusCode != http.StatusCreated {
		return smf_context.NupfCreateResult{}, classifyNupfHTTPStatus("create", response.StatusCode, body)
	}

	var created nupf.CreatedEventSubscription
	if err = json.Unmarshal(body, &created); err != nil {
		return smf_context.NupfCreateResult{}, malformedNupfSuccess("create", response.StatusCode)
	}
	result, err := validateNupfCreateSuccess(target, requestURI, response.Header.Get("Location"), created)
	if err != nil {
		return smf_context.NupfCreateResult{}, malformedNupfSuccess("create", response.StatusCode)
	}
	return result, nil
}

func (s *nupfEventExposureService) DeleteSubscription(
	ctx context.Context,
	target smf_context.EventExposureTarget,
	subscriptionID string,
) error {
	client := s.getEventExposureClient(target.APIroot)
	if client == nil {
		return nupfTransportError("delete")
	}

	callCtx, err := s.tokenContext(ctx)
	if err != nil {
		return &NupfEventExposureError{Kind: NupfEventExposureErrorToken, Operation: "delete"}
	}
	requestURI := strings.TrimRight(target.ServiceBaseURL, "/") +
		"/ee-subscriptions/" + url.PathEscape(subscriptionID)
	httpRequest, err := http.NewRequestWithContext(callCtx, http.MethodDelete, requestURI, nil)
	if err != nil {
		return nupfTransportError("delete")
	}
	httpRequest.Header.Set("Accept", "application/json")
	if err = authorizeNupfRequest(callCtx, httpRequest); err != nil {
		return &NupfEventExposureError{Kind: NupfEventExposureErrorToken, Operation: "delete"}
	}

	response, body, err := doNupfRequest(client, httpRequest)
	if err != nil {
		return classifyNupfHTTPError("delete", response, body, err)
	}
	if response.StatusCode != http.StatusNoContent {
		return classifyNupfHTTPStatus("delete", response.StatusCode, body)
	}
	return nil
}

func (s *nupfEventExposureService) tokenContext(ctx context.Context) (context.Context, error) {
	if !s.consumer.Context().OAuth2Required {
		return ctx, nil
	}

	tokenCtx, _, err := s.consumer.Context().GetTokenCtx(
		nupfServiceName, models.NrfNfManagementNfType_UPF)
	if err != nil {
		return nil, err
	}
	if source, ok := tokenCtx.Value(openapi.ContextOAuth2).(oauth2.TokenSource); ok {
		return context.WithValue(ctx, openapi.ContextOAuth2, source), nil
	}
	if token, ok := tokenCtx.Value(openapi.ContextAccessToken).(string); ok {
		return context.WithValue(ctx, openapi.ContextAccessToken, token), nil
	}
	return nil, errors.New("token context did not contain an access token")
}

func (s *nupfEventExposureService) createRequestURI(apiRoot string) string {
	s.EventExposureMu.RLock()
	defer s.EventExposureMu.RUnlock()
	return s.EventExposureCreateRequests[apiRoot]
}

func authorizeNupfRequest(ctx context.Context, request *http.Request) error {
	if source, ok := ctx.Value(openapi.ContextOAuth2).(oauth2.TokenSource); ok {
		token, err := source.Token()
		if err != nil {
			return err
		}
		token.SetAuthHeader(request)
		return nil
	}
	if token, ok := ctx.Value(openapi.ContextAccessToken).(string); ok && token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	return nil
}

func doNupfRequest(client *http.Client, request *http.Request) (*http.Response, []byte, error) {
	start := time.Now()
	response, err := client.Do(request)
	status := 0
	if response != nil {
		status = response.StatusCode
	}
	sbi_metrics.SbiMetricHook(
		request.Method, string(nupfServiceName), status, time.Since(start).Seconds())
	if response == nil {
		return nil, nil, err
	}
	defer response.Body.Close()
	body, readErr := io.ReadAll(io.LimitReader(response.Body, nupfEventExposureBodyLimit+1))
	if readErr != nil {
		return response, nil, readErr
	}
	if len(body) > nupfEventExposureBodyLimit {
		return response, nil, errors.New("nupf response body exceeded limit")
	}
	return response, body, err
}

func rejectNupfRedirects(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}

func classifyNupfHTTPError(
	operation string,
	response *http.Response,
	body []byte,
	err error,
) error {
	if response != nil {
		if response.StatusCode == http.StatusTemporaryRedirect ||
			response.StatusCode == http.StatusPermanentRedirect {
			return &NupfEventExposureError{
				Kind:       NupfEventExposureErrorRedirect,
				StatusCode: response.StatusCode,
				Operation:  operation,
			}
		}
		if response.StatusCode != 0 {
			return classifyNupfHTTPStatus(operation, response.StatusCode, body)
		}
	}
	_ = err
	return nupfTransportError(operation)
}

func classifyNupfHTTPStatus(operation string, status int, body []byte) error {
	classified := &NupfEventExposureError{
		Kind:       NupfEventExposureErrorUpstreamProblem,
		StatusCode: status,
		Operation:  operation,
	}
	var problem models.ProblemDetails
	if len(body) != 0 && json.Unmarshal(body, &problem) == nil && problem.Status != 0 {
		classified.ProblemDetails = &problem
	}
	if status == http.StatusTemporaryRedirect || status == http.StatusPermanentRedirect {
		classified.Kind = NupfEventExposureErrorRedirect
	}
	return classified
}

func malformedNupfSuccess(operation string, status int) error {
	return &NupfEventExposureError{
		Kind:       NupfEventExposureErrorMalformedSuccess,
		StatusCode: status,
		Operation:  operation,
	}
}

func nupfTransportError(operation string) error {
	return &NupfEventExposureError{
		Kind:      NupfEventExposureErrorTransport,
		Operation: operation,
	}
}

func validateNupfCreateSuccess(
	target smf_context.EventExposureTarget,
	requestURI string,
	location string,
	response nupf.CreatedEventSubscription,
) (smf_context.NupfCreateResult, error) {
	if response.SubscriptionID == "" || location == "" {
		return smf_context.NupfCreateResult{}, errors.New("missing subscription linkage")
	}

	resolvedLocation, err := resolveNupfLocation(requestURI, location)
	if err != nil {
		return smf_context.NupfCreateResult{}, err
	}
	if err = validateNupfLocation(target, resolvedLocation, response.SubscriptionID); err != nil {
		return smf_context.NupfCreateResult{}, err
	}

	return smf_context.NupfCreateResult{
		SubscriptionID:    response.SubscriptionID,
		ValidatedLocation: resolvedLocation.String(),
		CreateRequestURI:  requestURI,
		Response:          response,
		StatusCode:        http.StatusCreated,
	}, nil
}

func resolveNupfLocation(requestURI, location string) (*url.URL, error) {
	base, err := url.Parse(requestURI)
	if err != nil {
		return nil, err
	}
	ref, err := url.Parse(location)
	if err != nil {
		return nil, err
	}
	if hasNupfDotPathSegment(ref.Path) {
		return nil, errors.New("invalid location path")
	}
	resolved := base.ResolveReference(ref)
	if resolved.User != nil || resolved.RawQuery != "" || resolved.Fragment != "" {
		return nil, errors.New("invalid location metadata")
	}
	return resolved, nil
}

func validateNupfLocation(
	target smf_context.EventExposureTarget,
	location *url.URL,
	subscriptionID string,
) error {
	expectedBase, err := url.Parse(target.ServiceBaseURL)
	if err != nil {
		return err
	}
	if location.Scheme != expectedBase.Scheme || location.Host != expectedBase.Host {
		return errors.New("location origin mismatch")
	}

	if !validNupfSubscriptionIDSegment(subscriptionID) {
		return errors.New("invalid subscription id")
	}

	expectedPath := strings.TrimRight(expectedBase.EscapedPath(), "/") +
		"/ee-subscriptions/" + subscriptionID
	if location.EscapedPath() != expectedPath {
		return errors.New("location path mismatch")
	}
	return nil
}

func validNupfSubscriptionIDSegment(subscriptionID string) bool {
	return subscriptionID != "" &&
		subscriptionID != "." &&
		subscriptionID != ".." &&
		!strings.Contains(subscriptionID, "/") &&
		!strings.Contains(subscriptionID, `\`) &&
		url.PathEscape(subscriptionID) == subscriptionID
}

func hasNupfDotPathSegment(locationPath string) bool {
	for _, segment := range strings.Split(locationPath, "/") {
		if segment == "." || segment == ".." {
			return true
		}
	}
	return false
}
