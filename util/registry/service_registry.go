package registry

import "fmt"

// ServiceKey is a custom type for service registry keys.
type ServiceKey string

type ServiceRegistry interface {
	Register(key ServiceKey, svc interface{})
	Resolve(key ServiceKey) (interface{}, error)
}

type serviceRegistry struct {
	services map[ServiceKey]interface{}
}

func NewServiceRegistry() ServiceRegistry {
	return &serviceRegistry{
		services: make(map[ServiceKey]interface{}),
	}
}

func (r *serviceRegistry) Register(key ServiceKey, svc interface{}) {
	r.services[key] = svc
}

func (r *serviceRegistry) Resolve(key ServiceKey) (interface{}, error) {
	svc, ok := r.services[key]
	if !ok {
		return nil, fmt.Errorf("service not found: %s", key)
	}
	return svc, nil
}
