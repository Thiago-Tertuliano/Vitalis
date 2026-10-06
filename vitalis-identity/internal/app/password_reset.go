package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-identity/internal/port"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type PasswordResetService struct {
	users  port.UserRepository
	resets port.PasswordResetRepository
	hasher port.PasswordHasher
}

func NewPasswordResetService(users port.UserRepository, resets port.PasswordResetRepository, hasher port.PasswordHasher) *PasswordResetService {
	return &PasswordResetService{users: users, resets: resets, hasher: hasher}
}

func (s *PasswordResetService) Forgot(ctx context.Context, email string) (string, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	u, _, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		return "", nil // não vaza
	}
	token := uuid.NewString()
	if err := s.resets.Save(ctx, token, u.ID, time.Now().UTC().Add(time.Hour)); err != nil {
		return "", err
	}
	return token, nil
}

func (s *PasswordResetService) Redefinir(ctx context.Context, token, novaSenha string) error {
	if strings.TrimSpace(novaSenha) == "" {
		return fmt.Errorf("senha obrigatória")
	}
	userID, err := s.resets.Consume(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrResetInvalido
		}
		return err
	}
	hash, err := s.hasher.Hash(novaSenha)
	if err != nil {
		return err
	}
	return s.resets.UpdatePassword(ctx, userID, hash)
}