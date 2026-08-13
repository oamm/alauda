package storage

import (
	"context"
	"time"

	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
)

type AlertDispatcher interface {
	ProcessIncident(context.Context, *registryv1.Incident, string, time.Time) error
}
