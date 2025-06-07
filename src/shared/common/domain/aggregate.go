package domain

type Aggregate struct {
	domainEvents []DomainEvent
}

func (a *Aggregate) AddDomainEvent(dv DomainEvent) {
	if a.domainEvents == nil {
		a.domainEvents = make([]DomainEvent, 0)
	}
	a.domainEvents = append(a.domainEvents, dv)
}

func (a *Aggregate) GetDomainEvents() []DomainEvent {
	return a.domainEvents
}

func (a *Aggregate) ClearDomainEvents() {
	a.domainEvents = nil
}

func (a *Aggregate) PullDomainEvents() []DomainEvent {
	events := a.domainEvents
	a.domainEvents = nil
	return events
}

// type Aggregate[TId any] struct {
// 	Entity[TId]
// 	domainEvents []*DomainEvent
// }

// func (a *Aggregate[TId]) AddDomainEvent(dv *DomainEvent) {
// 	if a.domainEvents == nil {
// 		a.domainEvents = make([]*DomainEvent, 0)
// 	}
// 	a.domainEvents = append(a.domainEvents, dv)
// }

// func (a *Aggregate[TId]) GetDomainEvents() []*DomainEvent {
// 	return a.domainEvents
// }

// func (a *Aggregate[TId]) ClearDomainEvents() {
// 	a.domainEvents = nil
// }

// func (a *Aggregate[TId]) PullDomainEvents() []*DomainEvent {
// 	events := a.domainEvents
// 	a.domainEvents = nil
// 	return events
// }
