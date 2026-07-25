package processor

import (
	"context"
	"net"

	"github.com/google/uuid"

	"github.com/free5gc/smf/internal/compat/nupf"
	smf_context "github.com/free5gc/smf/internal/context"
	"github.com/free5gc/smf/internal/sbi/consumer"
	"github.com/free5gc/smf/pkg/app"
	"github.com/free5gc/smf/pkg/factory"
)

const (
	CONTEXT_NOT_FOUND = "CONTEXT_NOT_FOUND"
)

type ProcessorSmf interface {
	app.App

	Consumer() *consumer.Consumer
}

type Processor struct {
	ProcessorSmf

	eventExposure EventExposureDependencies
}

func NewProcessor(smf ProcessorSmf) (*Processor, error) {
	return NewProcessorWithEventExposureDependencies(smf, EventExposureDependencies{})
}

func NewProcessorWithEventExposureDependencies(
	smf ProcessorSmf,
	deps EventExposureDependencies,
) (*Processor, error) {
	if deps.Repository == nil {
		deps.Repository = smf_context.NewEventExposureRepository()
	}
	if deps.Resolver == nil {
		var err error
		deps.Resolver, err = configuredEventExposureResolver(factory.SmfConfig)
		if err != nil {
			return nil, err
		}
	}
	if deps.UUIDGenerator == nil {
		deps.UUIDGenerator = uuidGenerator{}
	}

	return &Processor{
		ProcessorSmf:  smf,
		eventExposure: deps,
	}, nil
}

func configuredEventExposureResolver(config *factory.Config) (EventExposureResolver, error) {
	if config == nil || config.Configuration == nil || config.Configuration.EventExposure == nil ||
		config.Configuration.EventExposure.StaticSessionResolution == nil ||
		!config.Configuration.EventExposure.StaticSessionResolution.Enabled {
		return smf_context.NewEventExposureTargetResolver(), nil
	}

	configured := config.Configuration.EventExposure.StaticSessionResolution.Sessions
	sessions := make([]smf_context.StaticEventExposureSession, 0, len(configured))
	for _, mapping := range configured {
		sessions = append(sessions, smf_context.StaticEventExposureSession{
			SUPI:         mapping.Supi,
			UEIPAddress:  net.ParseIP(mapping.UEIPv4).To4(),
			NupfAPIRoot:  mapping.NupfEeApiRoot,
			UPFName:      mapping.UPFName,
			Dnn:          mapping.Dnn,
			PDUSessionID: mapping.PDUSessionID,
		})
	}
	return smf_context.NewStaticEventExposureTargetResolver(sessions)
}

type EventExposureRepository interface {
	Store(smf_context.EventExposureSubscription) error
	Get(id string) (smf_context.EventExposureSubscription, bool)
	ClaimDelete(id string) (smf_context.EventExposureSubscription, bool)
}

type EventExposureResolver interface {
	ResolveEventExposureTarget(
		ctx context.Context,
		supi string,
		selectors smf_context.EventExposureSelectors,
	) (smf_context.EventExposureTarget, error)
}

type NupfEventExposureConsumer interface {
	CreateSubscription(
		ctx context.Context,
		target smf_context.EventExposureTarget,
		request nupf.CreateEventSubscription,
	) (smf_context.NupfCreateResult, error)
	DeleteSubscription(
		ctx context.Context,
		target smf_context.EventExposureTarget,
		subscriptionID string,
	) error
}

type UUIDGenerator interface {
	NewString() string
}

type EventExposureDependencies struct {
	Repository    EventExposureRepository
	Resolver      EventExposureResolver
	NupfConsumer  NupfEventExposureConsumer
	UUIDGenerator UUIDGenerator
}

type uuidGenerator struct{}

func (uuidGenerator) NewString() string {
	return uuid.NewString()
}
