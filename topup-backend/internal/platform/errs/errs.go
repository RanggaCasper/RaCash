// Package errs berisi tipe error domain yang dipetakan ke kode error kontrak
// (API_CONTRACT §1.4). Kode baru wajib ditambahkan ke kontrak lebih dulu.
package errs

import (
	"errors"
	"net/http"
)

type Code string

const (
	BadRequest               Code = "BAD_REQUEST"
	Unauthenticated          Code = "UNAUTHENTICATED"
	TokenExpired             Code = "TOKEN_EXPIRED"
	InvalidCredentials       Code = "INVALID_CREDENTIALS"
	InvalidSignature         Code = "INVALID_SIGNATURE"
	PermissionDenied         Code = "PERMISSION_DENIED"
	StepUpRequired           Code = "STEP_UP_REQUIRED"
	AccountSuspended         Code = "ACCOUNT_SUSPENDED"
	EmailNotVerified         Code = "EMAIL_NOT_VERIFIED"
	IPNotAllowed             Code = "IP_NOT_ALLOWED"
	NotFound                 Code = "NOT_FOUND"
	Conflict                 Code = "CONFLICT"
	InvalidStateTransition   Code = "INVALID_STATE_TRANSITION"
	IdempotencyConflict      Code = "IDEMPOTENCY_CONFLICT"
	VersionConflict          Code = "VERSION_CONFLICT"
	ValidationError          Code = "VALIDATION_ERROR"
	ProductUnavailable       Code = "PRODUCT_UNAVAILABLE"
	PaymentMethodUnavailable Code = "PAYMENT_METHOD_UNAVAILABLE"
	PromoInvalid             Code = "PROMO_INVALID"
	InsufficientBalance      Code = "INSUFFICIENT_BALANCE"
	AccountNotFound          Code = "ACCOUNT_NOT_FOUND"
	PriceChanged             Code = "PRICE_CHANGED"
	RateLimited              Code = "RATE_LIMITED"
	UpstreamError            Code = "UPSTREAM_ERROR"
	ServiceUnavailable       Code = "SERVICE_UNAVAILABLE"
	Internal                 Code = "INTERNAL_ERROR"
)

var catalog = map[Code]struct {
	status  int
	message string
}{
	BadRequest:               {http.StatusBadRequest, "Format request tidak valid."},
	Unauthenticated:          {http.StatusUnauthorized, "Silakan login terlebih dahulu."},
	TokenExpired:             {http.StatusUnauthorized, "Sesi telah kedaluwarsa."},
	InvalidCredentials:       {http.StatusUnauthorized, "Email atau password salah."},
	InvalidSignature:         {http.StatusUnauthorized, "Signature tidak valid."},
	PermissionDenied:         {http.StatusForbidden, "Anda tidak memiliki akses untuk aksi ini."},
	StepUpRequired:           {http.StatusForbidden, "Aksi ini memerlukan verifikasi 2FA ulang."},
	AccountSuspended:         {http.StatusForbidden, "Akun Anda dinonaktifkan."},
	EmailNotVerified:         {http.StatusForbidden, "Email belum diverifikasi."},
	IPNotAllowed:             {http.StatusForbidden, "IP tidak diizinkan."},
	NotFound:                 {http.StatusNotFound, "Data tidak ditemukan."},
	Conflict:                 {http.StatusConflict, "Data bentrok dengan data yang sudah ada."},
	InvalidStateTransition:   {http.StatusConflict, "Aksi tidak dapat dilakukan pada status saat ini."},
	IdempotencyConflict:      {http.StatusConflict, "Idempotency-Key sudah dipakai untuk request berbeda."},
	VersionConflict:          {http.StatusConflict, "Data sudah diubah pihak lain. Muat ulang lalu coba lagi."},
	ValidationError:          {http.StatusUnprocessableEntity, "Data yang dikirim tidak valid."},
	ProductUnavailable:       {http.StatusUnprocessableEntity, "Produk sedang tidak tersedia."},
	PaymentMethodUnavailable: {http.StatusUnprocessableEntity, "Metode pembayaran tidak tersedia."},
	PromoInvalid:             {http.StatusUnprocessableEntity, "Kode promo tidak berlaku."},
	InsufficientBalance:      {http.StatusUnprocessableEntity, "Saldo tidak mencukupi."},
	AccountNotFound:          {http.StatusUnprocessableEntity, "Akun tidak ditemukan."},
	PriceChanged:             {http.StatusUnprocessableEntity, "Harga telah berubah."},
	RateLimited:              {http.StatusTooManyRequests, "Terlalu banyak permintaan. Coba lagi nanti."},
	UpstreamError:            {http.StatusBadGateway, "Layanan mitra sedang bermasalah. Coba lagi nanti."},
	ServiceUnavailable:       {http.StatusServiceUnavailable, "Sistem sedang dalam pemeliharaan."},
	Internal:                 {http.StatusInternalServerError, "Terjadi kesalahan pada sistem."},
}

// Detail adalah informasi tambahan per field atau per aturan.
type Detail struct {
	Field              string `json:"field,omitempty"`
	Code               string `json:"code,omitempty"`
	Message            string `json:"message,omitempty"`
	RequiredPermission string `json:"required_permission,omitempty"`
}

// Error adalah error yang aman dikirim ke klien. Err (penyebab internal)
// hanya untuk log dan tidak pernah dikirim.
type Error struct {
	Code    Code
	Message string
	Details []Detail
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return string(e.Code) + ": " + e.Message + ": " + e.Err.Error()
	}
	return string(e.Code) + ": " + e.Message
}

func (e *Error) Unwrap() error { return e.Err }

// HTTPStatus mengembalikan status HTTP untuk kode error.
func (e *Error) HTTPStatus() int { return StatusOf(e.Code) }

// New membuat error dengan pesan default bila msg kosong.
func New(code Code, msg string) *Error {
	if msg == "" {
		msg = DefaultMessage(code)
	}
	return &Error{Code: code, Message: msg}
}

// Wrap membuat error dengan penyebab internal.
func Wrap(code Code, err error, msg string) *Error {
	e := New(code, msg)
	e.Err = err
	return e
}

// WithDetails mengembalikan salinan error dengan detail tambahan.
func (e *Error) WithDetails(d ...Detail) *Error {
	c := *e
	c.Details = append(append([]Detail(nil), e.Details...), d...)
	return &c
}

// Validation membuat VALIDATION_ERROR dengan detail per field.
func Validation(details ...Detail) *Error {
	return New(ValidationError, "").WithDetails(details...)
}

// Denied membuat PERMISSION_DENIED yang menyebut permission yang dibutuhkan.
func Denied(permission string) *Error {
	return New(PermissionDenied, "").WithDetails(Detail{RequiredPermission: permission})
}

// As mengekstrak *Error dari rantai error.
func As(err error) (*Error, bool) {
	var e *Error
	ok := errors.As(err, &e)
	return e, ok
}

// Is mengecek apakah err memiliki kode tertentu.
func Is(err error, code Code) bool {
	e, ok := As(err)
	return ok && e.Code == code
}

func StatusOf(code Code) int {
	if c, ok := catalog[code]; ok {
		return c.status
	}
	return http.StatusInternalServerError
}

func DefaultMessage(code Code) string {
	if c, ok := catalog[code]; ok {
		return c.message
	}
	return catalog[Internal].message
}
