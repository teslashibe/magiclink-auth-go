package fiberadapter

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/teslashibe/magiclink-auth-go"
)

// SendHandler handles POST /auth/magic-link.
func SendHandler(svc *magiclink.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req struct {
			Email string `json:"email"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
		}
		if err := svc.Send(c.UserContext(), req.Email); err != nil {
			return writeJSONError(c, err)
		}
		return c.JSON(fiber.Map{"status": "sent"})
	}
}

// VerifyCodeHandler handles POST /auth/verify.
func VerifyCodeHandler(svc *magiclink.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req struct {
			Email string `json:"email"`
			Code  string `json:"code"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
		}

		result, err := svc.VerifyCode(c.UserContext(), req.Email, req.Code)
		if err != nil {
			return writeJSONError(c, err)
		}
		return c.JSON(result)
	}
}

// VerifyLinkHandler handles GET /auth/verify?token=....
func VerifyLinkHandler(svc *magiclink.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := strings.TrimSpace(c.Query("token"))
		html, err := svc.VerifyTokenPage(c.UserContext(), token)
		if err != nil {
			return c.Status(magiclink.HTTPStatus(err)).SendString(magiclink.PublicError(err))
		}

		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.SendString(html)
	}
}

// AuthMiddleware validates bearer token, upserts user, and sets fiber locals:
// - user_id
// - magiclink_claims
func AuthMiddleware(svc *magiclink.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, claims, err := svc.AuthenticateBearer(c.UserContext(), c.Get("Authorization"))
		if err != nil {
			return writeJSONError(c, err)
		}
		c.Locals("user_id", userID)
		c.Locals("magiclink_claims", claims)
		return c.Next()
	}
}

func writeJSONError(c *fiber.Ctx, err error) error {
	return c.Status(magiclink.HTTPStatus(err)).JSON(fiber.Map{
		"error": magiclink.PublicError(err),
	})
}
