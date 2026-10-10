package httpx

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/RanggaCasper/RaCash/topup-backend/internal/platform/errs"
)

const (
	DefaultPerPage = 20
	MaxPerPage     = 100
)

type PageParams struct {
	Page    int
	PerPage int
}

func (p PageParams) Offset() int { return (p.Page - 1) * p.PerPage }
func (p PageParams) Limit() int  { return p.PerPage }

type PageMeta struct {
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	Total      int64 `json:"total"`
	TotalPages int64 `json:"total_pages"`
}

func NewPageMeta(p PageParams, total int64) PageMeta {
	pages := (total + int64(p.PerPage) - 1) / int64(p.PerPage)
	return PageMeta{Page: p.Page, PerPage: p.PerPage, Total: total, TotalPages: pages}
}

// ParsePage membaca ?page & ?per_page (default 1 & 20, maks. 100).
func ParsePage(c *fiber.Ctx) (PageParams, error) {
	p := PageParams{Page: 1, PerPage: DefaultPerPage}
	if v := c.Query("page"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return p, errs.Validation(errs.Detail{Field: "page", Code: "min", Message: "Minimal 1."})
		}
		p.Page = n
	}
	if v := c.Query("per_page"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > MaxPerPage {
			return p, errs.Validation(errs.Detail{Field: "per_page", Code: "max", Message: "Antara 1 dan 100."})
		}
		p.PerPage = n
	}
	return p, nil
}
