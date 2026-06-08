package rest

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/mailru/easyjson"
	"go.uber.org/zap"

	"scout/internal/interface/api/rest/dto"
)

// writeJSON marshals an easyjson DTO and writes it with the given status.
func writeJSON(c *fiber.Ctx, status int, v easyjson.Marshaler) error {
	b, err := easyjson.Marshal(v)
	if err != nil {
		return err
	}
	c.Status(status)
	c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	return c.Send(b)
}

// codeForStatus maps an HTTP status to a machine-readable error code.
func codeForStatus(status int) string {
	switch status {
	case fiber.StatusUnauthorized:
		return string(CodeAuth)
	case fiber.StatusForbidden:
		return string(CodeForbidden)
	case fiber.StatusNotFound:
		return string(CodeNotFound)
	case fiber.StatusBadRequest:
		return string(CodeBadRequest)
	default:
		if status >= fiber.StatusInternalServerError {
			return string(CodeInternal)
		}
		return string(CodeBadRequest)
	}
}

// ErrorHandler is the single place where errors are shaped into responses.
// It never leaks stack traces and always includes the correlation id.
func ErrorHandler(logger *zap.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		requestID, _ := c.Locals("request_id").(string)

		out := dto.ErrorDTO{RequestID: requestID}
		status := fiber.StatusInternalServerError

		var apiErr *APIError
		var fiberErr *fiber.Error
		switch {
		case errors.As(err, &apiErr):
			status = apiErr.Status
			out.Message = apiErr.Message
			out.Code = string(apiErr.Code)
			out.ResourceID = apiErr.ResourceID
			for _, d := range apiErr.Details {
				out.Details = append(out.Details, dto.FieldErrorDTO{Field: d.Field, Issue: d.Issue})
			}
		case errors.As(err, &fiberErr):
			status = fiberErr.Code
			out.Message = fiberErr.Message
			out.Code = codeForStatus(status)
		default:
			out.Message = "internal server error"
			out.Code = string(CodeInternal)
		}

		if status >= fiber.StatusInternalServerError {
			logger.Error("request failed",
				zap.String("request_id", requestID),
				zap.Int("status", status),
				zap.Error(err),
			)
		}
		return writeJSON(c, status, &out)
	}
}
