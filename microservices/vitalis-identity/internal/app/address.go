package app

import (
	"context"

	"github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-identity/internal/domain"
	"github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-identity/internal/port"
	"github.com/google/uuid"
)

type AddressService struct {
	repo port.AddressRepository
}

func NewAddressService(repo port.AddressRepository) *AddressService {
	return &AddressService{repo: repo}
}

func (s *AddressService) List(ctx context.Context, userID string) ([]domain.Address, error) {
	if userID == "" {
		return nil, ErrNaoAutenticado
	}
	return s.repo.ListByUser(ctx, userID)
}

func (s *AddressService) Create(ctx context.Context, userID string, a domain.Address) (domain.Address, error) {
	if userID == "" {
		return domain.Address{}, ErrNaoAutenticado
	}
	a.ID = uuid.NewString()
	a.UserID = userID
	if a.Label == "" {
		a.Label = "casa"
	}
	if err := s.repo.Create(ctx, a); err != nil {
		return domain.Address{}, err
	}
	return a, nil
}

func (s *AddressService) Update(ctx context.Context, userID string, a domain.Address) error {
	if userID == "" {
		return ErrNaoAutenticado
	}
	a.UserID = userID
	return s.repo.Update(ctx, a)
}

func (s *AddressService) Delete(ctx context.Context, userID, id string) error {
	if userID == "" {
		return ErrNaoAutenticado
	}
	return s.repo.Delete(ctx, userID, id)
}