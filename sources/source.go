package sources

import (
	"context"
	"fmt"
	"github.com/kubemq-hub/kubemq-bridges/config"
	"github.com/kubemq-hub/kubemq-bridges/middleware"
	"github.com/kubemq-hub/kubemq-bridges/pkg/logger"
	"github.com/kubemq-hub/kubemq-bridges/sources/command"
	"github.com/kubemq-hub/kubemq-bridges/sources/events"
	events_store "github.com/kubemq-hub/kubemq-bridges/sources/events-store"
	"github.com/kubemq-hub/kubemq-bridges/sources/query"
	"github.com/kubemq-hub/kubemq-bridges/sources/queue"
)

type Source interface {
	Init(ctx context.Context, connection config.Metadata) error
	Start(ctx context.Context, target []middleware.Middleware, log *logger.Logger) error
	Stop() error
}

func Init(ctx context.Context, kind string, connection config.Metadata) (Source, error) {
	switch kind {
	case "source.command":
		source := command.New()
		if err := source.Init(ctx, connection); err != nil {
			return nil, err
		}
		return source, nil
	case "source.query":
		source := query.New()
		if err := source.Init(ctx, connection); err != nil {
			return nil, err
		}
		return source, nil
	case "source.events":
		source := events.New()
		if err := source.Init(ctx, connection); err != nil {
			return nil, err
		}
		return source, nil
	case "source.events-store":
		source := events_store.New()
		if err := source.Init(ctx, connection); err != nil {
			return nil, err
		}
		return source, nil
	case "source.queue":
		source := queue.New()
		if err := source.Init(ctx, connection); err != nil {
			return nil, err
		}
		return source, nil

	default:
		return nil, fmt.Errorf("invalid kind %s for source", kind)
	}

}
