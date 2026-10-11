// Package token menerbitkan dan memverifikasi access token JWT (HS256).
package token

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/modules/auth/internal/domain"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/clock"
	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/errs"
)

// Issuer menandatangani dan memverifikasi access token.
type Issuer struct {
	secret []byte
	clock  clock.Clock
}

func NewIssuer(secret string, clk clock.Clock) *Issuer {
	return &Issuer{secret: []byte(secret), clock: clk}
}

// wire adalah bentuk claim di JWT: sub, sid, pv, iat, exp (API_CONTRACT §1.7).
type wire struct {
	SID string `json:"sid"`
	PV  int32  `json:"pv"`
	// Lim membatasi token ke setup 2FA (domain.LimitSetup2FA); kosong = akses penuh.
	Lim string `json:"lim,omitempty"`
	jwt.RegisteredClaims
}

// Sign membuat access token untuk sesi. restricted=true membatasi token ke
// endpoint setup 2FA. Mengembalikan token dan waktu kedaluwarsanya.
func (i *Issuer) Sign(userID, sessionID uuid.UUID, permVersion int32, restricted bool) (string, time.Time, error) {
	now := i.clock.Now()
	exp := now.Add(domain.AccessTokenTTL)
	w := wire{
		SID: sessionID.String(),
		PV:  permVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}
	if restricted {
		w.Lim = domain.LimitSetup2FA
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, w).SignedString(i.secret)
	return signed, exp, err
}

// Parse memverifikasi tanda tangan, algoritma (hanya HS256), dan masa berlaku.
// Token kedaluwarsa menghasilkan errs.TokenExpired; token rusak atau palsu
// menghasilkan errs.Unauthenticated.
func (i *Issuer) Parse(raw string) (domain.Claims, error) {
	var w wire
	_, err := jwt.ParseWithClaims(raw, &w,
		func(*jwt.Token) (any, error) { return i.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(i.clock.Now),
	)
	switch {
	case errors.Is(err, jwt.ErrTokenExpired):
		return domain.Claims{}, errs.New(errs.TokenExpired, "")
	case err != nil:
		return domain.Claims{}, errs.New(errs.Unauthenticated, "")
	}
	uid, err1 := uuid.Parse(w.Subject)
	sid, err2 := uuid.Parse(w.SID)
	if err1 != nil || err2 != nil || w.IssuedAt == nil {
		return domain.Claims{}, errs.New(errs.Unauthenticated, "")
	}
	return domain.Claims{
		UserID: uid, SessionID: sid, PermVersion: w.PV, Restricted: w.Lim == domain.LimitSetup2FA,
		IssuedAt: w.IssuedAt.Time, ExpiresAt: w.ExpiresAt.Time,
	}, nil
}
