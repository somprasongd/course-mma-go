package domain

import "time"

type EventName string
type DomainEvent interface {
	EventName() EventName
	OccurredAt() time.Time
}

type BaseDomainEvent struct {
	Name EventName
	At   time.Time
}

// ไม่ใช้ pointer receiver
// Read-only (ไม่มีการเปลี่ยนค่า)
// Struct ขนาดเล็ก

func (e BaseDomainEvent) EventName() EventName {
	return e.Name
}

func (e BaseDomainEvent) OccurredAt() time.Time {
	return e.At
}
