package rest

import (
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"scout/internal/application/authapp"
	"scout/internal/interface/api/rest/dto"
	"scout/internal/interface/api/rest/validator"
)

// AuthController handles authentication endpoints.
type AuthController struct {
	logger *zap.Logger
	auth   *authapp.Service
}

// NewAuthController builds an AuthController.
func NewAuthController(logger *zap.Logger, auth *authapp.Service) *AuthController {
	return &AuthController{logger: logger, auth: auth}
}

// Login authenticates a user and returns a JWT.
func (a *AuthController) Login(c *fiber.Ctx) error {
	var req dto.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return NewBadRequest("invalid JSON body")
	}
	if errs := validator.ValidateLogin(req); len(errs) > 0 {
		return NewValidation("invalid request body", toFieldErrors(errs))
	}
	token, err := a.auth.Login(c.Context(), req.Login, req.Password)
	if err != nil {
		if authapp.IsInvalidCredentials(err) {
			return NewAuth("invalid login or password")
		}
		return err
	}
	return writeJSON(c, fiber.StatusOK, &dto.LoginResponseDTO{AccessToken: token, TokenType: "Bearer"})
}

// toFieldErrors converts validator field errors into the rest error shape.
func toFieldErrors(in []validator.FieldError) []FieldError {
	out := make([]FieldError, len(in))
	for i, e := range in {
		out[i] = FieldError{Field: e.Field, Issue: e.Issue}
	}
	return out
}
