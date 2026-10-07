package app

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-identity/internal/domain"
	"github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-identity/internal/outbox"
	"github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-identity/internal/port"
	"github.com/google/uuid"
)

type ProfileService struct {
	users  port.UserRepository
	outbox *outbox.Writer
}

func NewProfileService(users port.UserRepository, ob *outbox.Writer) *ProfileService {
	return &ProfileService{users: users, outbox: ob}
}

func (s *ProfileService) Me(ctx context.Context, userID string) (domain.User, error) {
	if userID == "" {
		return domain.User{}, ErrNaoAutenticado
	}
	return s.users.FindByID(ctx, userID)
}

type UpdateProfileInput struct {
	Nome           string
	Telefone       string
	CPF            string
	DataNascimento *time.Time
	CorrelationID  string
}

func (s *ProfileService) Update(ctx context.Context, userID string, in UpdateProfileInput) (domain.User, error) {
	if userID == "" {
		return domain.User{}, ErrNaoAutenticado
	}
	u, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return domain.User{}, err
	}
	u.Nome = in.Nome
	u.Telefone = in.Telefone
	u.CPF = in.CPF
	u.DataNascimento = in.DataNascimento
	if err := s.users.UpdateProfile(ctx, u); err != nil {
		return domain.User{}, err
	}
	payload, _ := json.Marshal(map[string]any{"user_id": u.ID})
	if s.outbox != nil {
		_ = s.outbox.Enqueue(ctx, outbox.Event{
			ID: uuid.NewString(), Type: "user.updated", Producer: producer,
			Version: 1, CorrelationID: in.CorrelationID, Data: payload,
		})
	}
	return s.users.FindByID(ctx, userID)
}
