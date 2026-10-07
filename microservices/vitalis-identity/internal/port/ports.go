package port

import (
	"context"
	"time"

	"github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-identity/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context, u domain.User, passwordHash string) error
	FindByEmail(ctx context.Context, email string) (domain.User, string, error)
	FindByID(ctx context.Context, id string) (domain.User, error)
	UpdateProfile(ctx context.Context, u domain.User) error
	ReplaceRoles(ctx context.Context, userID string, roles []string) error
	EmailExists(ctx context.Context, email string) (bool, error)
}

type RefreshTokenRepository interface {
	Save(ctx context.Context, jti, userID string, expiresAt time.Time) error
	FindValid(ctx context.Context, jti string) (userID string, err error)
	Revoke(ctx context.Context, jti string) error
}
type AddressRepository interface {
	ListByUser(ctx context.Context, userID string) ([]domain.Address, error)
	Create(ctx context.Context, a domain.Address) error
	Update(ctx context.Context, a domain.Address) error
	Delete(ctx context.Context, userID, id string) error
}
type PasswordResetRepository interface {
	Save(ctx context.Context, token, userID string, expiresAt time.Time) error
	Consume(ctx context.Context, token string) (userID string, err error)
	UpdatePassword(ctx context.Context, userID, hash string) error
}
type PasswordHasher interface {
	Hash(plain string) (string, error)
	Compare(hash, plain string) bool
}
type TokenIssuer interface {
	IssueAccess(user domain.User) (token string, expiresInSec int, err error)
	IssueRefresh(ctx context.Context, userID string) (refreshToken string, err error)
	ParseRefresh(token string) (jti, userID string, err error)
	PublicJWKS() (map[string]any, error)
}
type Clock interface{ Now() time.Time }
type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now().UTC() }
