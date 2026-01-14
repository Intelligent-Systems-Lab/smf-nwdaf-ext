package consumer

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/free5gc/openapi/models"
	"github.com/free5gc/openapi/nwdaf/EventsSubscription"
	sbi_metrics "github.com/free5gc/util/metrics/sbi"
)

type nwdafService struct {
	consumer *Consumer

	mu sync.RWMutex

	EventsSubscriptionClients map[string]*EventsSubscription.APIClient
}

func (s *nwdafService) getEventsSubscriptionClient(apiRoot string) *EventsSubscription.APIClient {
	if apiRoot == "" {
		return nil
	}

	basePath := normalizeNwdafBasePath(apiRoot)
	s.mu.RLock()
	client, ok := s.EventsSubscriptionClients[basePath]
	if ok {
		s.mu.RUnlock()
		return client
	}

	configuration := EventsSubscription.NewConfiguration()
	configuration.SetBasePath(basePath)
	configuration.SetMetrics(sbi_metrics.SbiMetricHook)
	client = EventsSubscription.NewAPIClient(configuration)

	s.mu.RUnlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.EventsSubscriptionClients[basePath] = client
	return client
}

func (s *nwdafService) SendCreateNwdafEventsSubscription(
	ctx context.Context,
	apiRoot string,
	subscription *models.NnwdafEventsSubscription,
) (string, *models.NnwdafEventsSubscription, error) {
	client := s.getEventsSubscriptionClient(apiRoot)
	if client == nil {
		return "", nil, fmt.Errorf("nwdaf api root is empty")
	}

	request := &EventsSubscription.CreateNWDAFEventsSubscriptionRequest{
		NnwdafEventsSubscription: subscription,
	}
	response, err := client.NWDAFEventsSubscriptionsCollectionApi.CreateNWDAFEventsSubscription(ctx, request)
	if err != nil {
		return "", nil, err
	}

	return response.Location, &response.NnwdafEventsSubscription, nil
}

func (s *nwdafService) SendDeleteNwdafEventsSubscription(
	ctx context.Context,
	apiRoot string,
	subscriptionId string,
) error {
	client := s.getEventsSubscriptionClient(apiRoot)
	if client == nil {
		return fmt.Errorf("nwdaf api root is empty")
	}

	request := &EventsSubscription.DeleteNWDAFEventsSubscriptionRequest{
		SubscriptionId: subscriptionId,
	}
	_, err := client.IndividualNWDAFEventsSubscriptionDocumentApi.DeleteNWDAFEventsSubscription(ctx, request)
	return err
}

func normalizeNwdafBasePath(apiRoot string) string {
	trimmed := strings.TrimRight(apiRoot, "/")
	if strings.Contains(trimmed, "/nnwdaf-eventssubscription/") {
		return trimmed
	}
	return trimmed + "/nnwdaf-eventssubscription/v1"
}
