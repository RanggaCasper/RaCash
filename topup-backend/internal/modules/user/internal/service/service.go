// Package service berisi use case modul user.
package service

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/modules/user/internal/domain"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/modules/user/internal/repository"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/clock"
)

type Service struct {
	repo  *repository.Repo
	clock clock.Clock
}

func New(repo *repository.Repo, clk clock.Clock) *Service {
	return &Service{repo: repo, clock: clk}
}

// Create membuat user PENDING_VERIFICATION dengan tier default. Email
// dinormalisasi ke huruf kecil (constraint users_email_lower_chk).
func (s *Service) Create(ctx context.Context, in domain.NewUser) (domain.User, error) {
	in.Email = NormalizeEmail(in.Email)
	userID, err := s.repo.Create(ctx, in, s.clock.Now())
	if err != nil {
		return domain.User{}, err
	}
	c, err := s.repo.FindByID(ctx, userID)
	return c.User, err
}

func (s *Service) Get(ctx context.Context, userID uuid.UUID) (domain.User, error) {
	c, err := s.repo.FindByID(ctx, userID)
	return c.User, err
}

func (s *Service) FindByEmail(ctx context.Context, email string) (domain.User, error) {
	c, err := s.repo.FindByEmail(ctx, NormalizeEmail(email))
	return c.User, err
}

func (s *Service) CredentialsByEmail(ctx context.Context, email string) (domain.Credentials, error) {
	return s.repo.FindByEmail(ctx, NormalizeEmail(email))
}

func (s *Service) MarkEmailVerified(ctx context.Context, userID uuid.UUID) error {
	return s.repo.MarkEmailVerified(ctx, userID, s.clock.Now())
}

func (s *Service) SetPasswordHash(ctx context.Context, userID uuid.UUID, hash string) error {
	return s.repo.SetPasswordHash(ctx, userID, hash, s.clock.Now())
}

func (s *Service) SetTwoFactorEnabled(ctx context.Context, userID uuid.UUID, enabled bool) error {
	return s.repo.SetTwoFactorEnabled(ctx, userID, enabled, s.clock.Now())
}

func (s *Service) RecordLogin(ctx context.Context, userID uuid.UUID) error {
	return s.repo.TouchLastLogin(ctx, userID, s.clock.Now())
}

// NormalizeEmail memangkas spasi dan menjadikan huruf kecil.
func NormalizeEmail(email string) string { return strings.ToLower(strings.TrimSpace(email)) }
