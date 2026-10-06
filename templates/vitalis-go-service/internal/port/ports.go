package port

import (
	"context"
	"time"

	"github.com/Rede-Medica-D-Excelencia-Vitalis/vitalis-service-template/internal/domain"
)

// ExampleRepository — porta de persistência (hexagonal).
type ExampleRepository interface {
	Save(ctx context.Context, e domain.Example) error
	FindByID(ctx context.Context, id string) (domain.Example, error)
}

// EventPublisher — publica no barramento (Redis Streams / via outbox worker).
type EventPublisher interface {
	Publish(ctx context.Context, stream string, values map[string]interface{}) error
}

// Clock abstrai time.Now para testes.
type Clock interface {
	Now() time.Time
}

type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now().UTC() }
