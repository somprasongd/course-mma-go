package registry

import "fmt"

// ServiceKey is a custom type for service registry keys.
type ServiceKey string

type ProvidedService struct {
	Key   ServiceKey
	Value any
}

type ServiceRegistry interface {
	Register(key ServiceKey, svc any)
	Resolve(key ServiceKey) (any, error)
}

type serviceRegistry struct {
	services map[ServiceKey]any
}

func NewServiceRegistry() ServiceRegistry {
	return &serviceRegistry{
		services: make(map[ServiceKey]any),
	}
}

func (r *serviceRegistry) Register(key ServiceKey, svc any) {
	r.services[key] = svc
}

func (r *serviceRegistry) Resolve(key ServiceKey) (any, error) {
	svc, ok := r.services[key]
	if !ok {
		return nil, fmt.Errorf("service not found: %s", key)
	}
	return svc, nil
}
