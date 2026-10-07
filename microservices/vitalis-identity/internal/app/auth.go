package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-identity/internal/domain"
	"github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-identity/internal/outbox"
	"github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-identity/internal/port"
	"github.com/google/uuid"
)

const producer = "vitalis-identity"

type AuthService struct {
	users   port.UserRepository
	refresh port.RefreshTokenRepository
	hasher  port.PasswordHasher
	tokens  port.TokenIssuer
	outbox  *outbox.Writer
	clock   port.Clock
}

func NewAuthService(
	users port.UserRepository,
	refresh port.RefreshTokenRepository,
	hasher port.PasswordHasher,
	tokens port.TokenIssuer,
	ob *outbox.Writer,
	clock port.Clock,
) *AuthService {
	if clock == nil {
		clock = port.RealClock{}
	}
	return &AuthService{users: users, refresh: refresh, hasher: hasher, tokens: tokens, outbox: ob, clock: clock}
}

type RegisterInput struct {
	Nome           string
	Email          string
	Senha          string
	Telefone       string
	TipoUsuario    string
	CPF            string
	DataNascimento *time.Time
	CorrelationID  string
}

type UserDTO struct {
	ID    string   `json:"id"`
	Nome  string   `json:"nome"`
	Email string   `json:"email"`
	Roles []string `json:"roles"`
}

type TokenOutput struct {
	AccessToken  string  `json:"access_token"`
	RefreshToken string  `json:"refresh_token,omitempty"`
	TokenType    string  `json:"token_type"`
	ExpiresIn    int     `json:"expires_in"`
	User         UserDTO `json:"user"`
}

func (s *AuthService) Register(ctx context.Context, in RegisterInput) (UserDTO, error) {
	in.Email = strings.TrimSpace(strings.ToLower(in.Email))
	if in.Nome == "" || in.Email == "" || in.Senha == "" {
		return UserDTO{}, fmt.Errorf("nome, email e senha são obrigatórios")
	}
	if !domain.RoleValida(in.TipoUsuario) {
		return UserDTO{}, ErrRoleInvalida
	}
	exists, err := s.users.EmailExists(ctx, in.Email)
	if err != nil {
		return UserDTO{}, err
	}
	if exists {
		return UserDTO{}, ErrEmailJaExiste
	}

	hash, err := s.hasher.Hash(in.Senha)
	if err != nil {
		return UserDTO{}, err
	}

	now := s.clock.Now()
	u := domain.User{
		ID:             uuid.NewString(),
		Email:          in.Email,
		Nome:           in.Nome,
		Telefone:       in.Telefone,
		Status:         domain.StatusActive, // MVP: já ativo
		CPF:            in.CPF,
		DataNascimento: in.DataNascimento,
		Roles:          []string{in.TipoUsuario},
		CreatedAt:      now,
		UpdateAt:       now,
	}
	if err := s.users.Create(ctx, u, hash); err != nil {
		return UserDTO{}, err
	}

	payload, _ := json.Marshal(map[string]any{
		"user_id": u.ID,
		"email":   u.Email,
		"roles":   u.Roles,
	})
	if s.outbox != nil {
		_ = s.outbox.Enqueue(ctx, outbox.Event{
			ID:            uuid.NewString(),
			Type:          "user.registered",
			Producer:      producer,
			Version:       1,
			CorrelationID: in.CorrelationID,
			Data:          payload,
		})
	}

	return UserDTO{ID: u.ID, Nome: u.Nome, Email: u.Email, Roles: u.Roles}, nil
}

type LoginInput struct {
	Email string
	Senha string
}

func (s *AuthService) Login(ctx context.Context, in LoginInput) (TokenOutput, error) {
	email := strings.TrimSpace(strings.ToLower(in.Email))
	user, hash, err := s.users.FindByEmail(ctx, email)
	if err != nil || !s.hasher.Compare(hash, in.Senha) {
		// mensagem genérica (não vaza se email existe)
		return TokenOutput{}, ErrCredenciaisInvalidas
	}
	if !user.PodeReceberToken() {
		return TokenOutput{}, ErrUsuarioInativo
	}

	access, exp, err := s.tokens.IssueAccess(user)
	if err != nil {
		return TokenOutput{}, err
	}
	refresh, err := s.tokens.IssueRefresh(ctx, user.ID)
	if err != nil {
		return TokenOutput{}, err
	}

	return TokenOutput{
		AccessToken:  access,
		RefreshToken: refresh,
		TokenType:    "Bearer",
		ExpiresIn:    exp,
		User:         UserDTO{ID: user.ID, Nome: user.Nome, Email: user.Email, Roles: user.Roles},
	}, nil
}

type RefreshInput struct {
	RefreshToken string
}

func (s *AuthService) Refresh(ctx context.Context, in RefreshInput) (TokenOutput, error) {
	jti, userID, err := s.tokens.ParseRefresh(in.RefreshToken)
	if err != nil {
		return TokenOutput{}, ErrRefreshInvalido
	}
	if _, err := s.refresh.FindValid(ctx, jti); err != nil {
		return TokenOutput{}, ErrRefreshInvalido
	}
	// rotação: revoga o antigo
	_ = s.refresh.Revoke(ctx, jti)

	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return TokenOutput{}, ErrRefreshInvalido
	}
	if !user.PodeReceberToken() {
		return TokenOutput{}, ErrUsuarioInativo
	}

	access, exp, err := s.tokens.IssueAccess(user)
	if err != nil {
		return TokenOutput{}, err
	}
	newRefresh, err := s.tokens.IssueRefresh(ctx, user.ID)
	if err != nil {
		return TokenOutput{}, err
	}
	return TokenOutput{
		AccessToken:  access,
		RefreshToken: newRefresh,
		TokenType:    "Bearer",
		ExpiresIn:    exp,
		User:         UserDTO{ID: user.ID, Nome: user.Nome, Email: user.Email, Roles: user.Roles},
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	jti, _, err := s.tokens.ParseRefresh(refreshToken)
	if err != nil {
		return nil // idempotente
	}
	_ = s.refresh.Revoke(ctx, jti)
	return nil
}

func (s *AuthService) JWKS() (map[string]any, error) {
	return s.tokens.PublicJWKS()
}
