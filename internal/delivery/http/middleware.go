package http

import (
	"net/http"
	"strings"

	"github.com/FilipeAphrody/sentinel-auth/pkg/security"
	"github.com/labstack/echo/v4"
)

// JWTMiddleware intercepts the request to validate the JWT token in the Authorization header.
func JWTMiddleware(secret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) error {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return c.JSON(http.StatusUnauthorized, echo.Map{"error": "missing authorization header"})
			}

			// Expected format: "Bearer <token>"
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				return c.JSON(http.StatusUnauthorized, echo.Map{"error": "invalid authorization format"})
			}

			// Validate token using the security package logic
			claims, err := security.ValidateToken(parts[1], secret)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, echo.Map{"error": "invalid or expired token"})
			}

			// Inject extracted user information into Echo context.
			// This allows subsequent handlers to identify the user.
			c.Set("user_id", claims.UserID)
			c.Set("role", claims.Role)

			return next(c)
		}
	}
}

// RoleMiddleware ensures only users with specific roles (or admins) can access the route.
func RoleMiddleware(requiredRole string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) error {
		return func(c echo.Context) error {
			role, ok := c.Get("role").(string)
			
			// Authorization logic: Admins have full access, others need the specific role.
			if !ok || (role != requiredRole && role != "admin") {
				return c.JSON(http.StatusForbidden, echo.Map{"error": "access denied: insufficient permissions"})
			}
			
			return next(c)
		}
	}
}