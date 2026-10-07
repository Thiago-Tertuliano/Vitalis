package app

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-identity/internal/domain"
	"github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-identity/internal/outbox"
	"github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-identity/internal/port"
	"github.com/google/uuid"
)

type AdminService struct {
	users  port.UserRepository
	outbox *outbox.Writer
}

func NewAdminService(users port.UserRepository, ob *outbox.Writer) *AdminService {
	return &AdminService{users: users, outbox: ob}
}

func (s *AdminService) ReplaceRoles(ctx context.Context, actorRoles string, targetUserID string, roles []string, correlationID string) error {
	if !strings.Contains(actorRoles, "admin") {
		return ErrForbidden
	}
	for _, r := range roles {
		if !domain.RoleValida(r) {
			return ErrRoleInvalida
		}
	}
	if err := s.users.ReplaceRoles(ctx, targetUserID, roles); err != nil {
		return err
	}
	payload, _ := json.Marshal(map[string]any{"user_id": targetUserID, "roles": roles})
	if s.outbox != nil {
		_ = s.outbox.Enqueue(ctx, outbox.Event{
			ID: uuid.NewString(), Type: "user.role_changed", Producer: producer,
			Version: 1, CorrelationID: correlationID, Data: payload,
		})
	}
	return nil
}
