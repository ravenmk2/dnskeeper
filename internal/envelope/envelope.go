package envelope

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/ravenmk2/dnskeeper/internal/apperr"
)

type Envelope struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Error   *ErrorBody  `json:"error"`
}

type ErrorBody struct {
	Code     string         `json:"code"`
	Message  string         `json:"message"`
	Details  []apperr.Detail `json:"details,omitempty"`
}

func Data(c echo.Context, data interface{}) error {
	return c.JSON(http.StatusOK, Envelope{
		Success: true,
		Data:    data,
		Error:   nil,
	})
}

func Paged(c echo.Context, items interface{}, page, size, total int) error {
	pageCount := 0
	if size > 0 {
		pageCount = (total + size - 1) / size
	}
	return c.JSON(http.StatusOK, Envelope{
		Success: true,
		Data: map[string]interface{}{
			"items":      items,
			"page":       page,
			"size":       size,
			"total":      total,
			"page_count": pageCount,
		},
		Error: nil,
	})
}

func Error(c echo.Context, code int, e *apperr.AppError) error {
	return c.JSON(code, Envelope{
		Success: false,
		Data:    nil,
		Error: &ErrorBody{
			Code:    e.Code,
			Message: e.Message,
			Details: e.Details,
		},
	})
}
