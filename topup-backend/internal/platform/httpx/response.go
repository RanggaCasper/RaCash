// Package httpx: envelope respons, paginasi, request id, binding, dan error
// handler Fiber sesuai API_CONTRACT §1.3–§1.5.
package httpx

import (
	"github.com/gofiber/fiber/v2"
)

// Prefix grup route (API_CONTRACT §1.1).
const (
	PrefixPublic  = "/api/v1"
	PrefixAdmin   = "/api/admin/v1"
	PrefixH2H     = "/api/h2h/v1"
	PrefixWebhook = "/api/v1/webhooks"
)

type envelope struct {
	Success   bool      `json:"success"`
	Data      any       `json:"data"`
	Meta      *PageMeta `json:"meta,omitempty"`
	RequestID string    `json:"request_id"`
}

// OK mengirim 200 dengan data.
func OK(c *fiber.Ctx, data any) error { return send(c, fiber.StatusOK, data, nil) }

// Created mengirim 201 dengan data.
func Created(c *fiber.Ctx, data any) error { return send(c, fiber.StatusCreated, data, nil) }

// Accepted mengirim 202 dengan data (mis. job asinkron).
func Accepted(c *fiber.Ctx, data any) error { return send(c, fiber.StatusAccepted, data, nil) }

// Page mengirim daftar berhalaman dengan meta.
func Page(c *fiber.Ctx, data any, meta PageMeta) error {
	return send(c, fiber.StatusOK, data, &meta)
}

func send(c *fiber.Ctx, status int, data any, meta *PageMeta) error {
	return c.Status(status).JSON(envelope{Success: true, Data: data, Meta: meta, RequestID: RequestID(c)})
}
