package http

import "github.com/gofiber/fiber/v2"

const apiKeyHeader = "X-API-Key"

// auth creates API-key middleware for private routes.
func auth(accessKey string) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		if ctx.Get(apiKeyHeader) != accessKey {
			return ctx.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
				Error: "unauthorized",
			})
		}

		return ctx.Next()
	}
}
