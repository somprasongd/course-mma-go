package eventbus

import (
	"time"
)

type EventName string

type Event interface {
	EventID() string       // UUID หรือ ULID
	EventName() EventName  // เช่น "CustomerCreated"
	OccurredAt() time.Time // เวลาที่ event เกิด
}

type BaseEvent struct {
	ID   string
	Name EventName
	At   time.Time
}

func (e BaseEvent) EventID() string {
	return e.ID
}

func (e BaseEvent) EventName() EventName {
	return e.Name
}

func (e BaseEvent) OccurredAt() time.Time {
	return e.At
}
