package app

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Rede-Medica-D-Excelencia-Vitalis/vitalis-service-template/internal/domain"
	"github.com/Rede-Medica-D-Excelencia-Vitalis/vitalis-service-template/internal/outbox"
	"github.com/Rede-Medica-D-Excelencia-Vitalis/vitalis-service-template/internal/port"
	"github.com/google/uuid"
)

// ExampleService — caso de uso de exemplo (substitua pelos use cases do domínio).
type ExampleService struct {
	repo   port.ExampleRepository
	outbox *outbox.Writer
	clock  port.Clock
}

func NewExampleService(repo port.ExampleRepository, ob *outbox.Writer, clock port.Clock) *ExampleService {
	if clock == nil {
		clock = port.RealClock{}
	}
	return &ExampleService{repo: repo, outbox: ob, clock: clock}
}

type CreateExampleInput struct {
	Name          string
	CorrelationID string
}

func (s *ExampleService) Create(ctx context.Context, in CreateExampleInput) (domain.Example, error) {
	if in.Name == "" {
		return domain.Example{}, fmt.Errorf("name é obrigatório")
	}
	e := domain.Example{
		ID:        uuid.NewString(),
		Name:      in.Name,
		CreatedAt: s.clock.Now(),
	}
	if err := s.repo.Save(ctx, e); err != nil {
		return domain.Example{}, err
	}

	payload, _ := json.Marshal(map[string]any{
		"example_id": e.ID,
		"name":       e.Name,
	})
	if s.outbox != nil {
		if err := s.outbox.Enqueue(ctx, outbox.Event{
			ID:            uuid.NewString(),
			Type:          "example.created",
			Producer:      "vitalis-service-template",
			Version:       1,
			CorrelationID: in.CorrelationID,
			Data:          payload,
		}); err != nil {
			return domain.Example{}, fmt.Errorf("outbox: %w", err)
		}
	}
	return e, nil
}
