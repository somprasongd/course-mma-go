package eventbus

import (
	"context"
)

type IntegrationEventHandler interface {
	Handle(ctx context.Context, event Event) error
}

type EventBus interface {
	Publish(ctx context.Context, event Event) error
	Subscribe(eventName EventName, handler IntegrationEventHandler)
}
