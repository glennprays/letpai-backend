package middleware

import (
	"github.com/glennprays/letpai-backend/internal/service"
	"github.com/gofiber/fiber/v2"
)

// AdminRoles defines valid admin roles
const (
	AdminRoleAdmin      = "admin"
	AdminRoleSuperAdmin = "super_admin"
)

// RequireAdminRole creates middleware that requires admin role
func RequireAdminRole(jwtSvc *service.JWTService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get user ID from context (should be set by Authenticate middleware)
		userID := GetUserID(c)
		if userID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "AUTH_002",
					"message": "Authentication required",
				},
			})
		}

		// Get role from context (should be set by updated Authenticate middleware)
		role, ok := c.Locals(roleKey).(string)
		if !ok || role == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "AUTH_002",
					"message": "Role not found in token",
				},
			})
		}

		// Check if user has admin role
		if role != AdminRoleAdmin && role != AdminRoleSuperAdmin {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "AUTH_003",
					"message": "Admin access required",
				},
			})
		}

		return c.Next()
	}
}

// RequireSuperAdminRole creates middleware that requires super_admin role
func RequireSuperAdminRole(jwtSvc *service.JWTService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get user ID from context
		userID := GetUserID(c)
		if userID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "AUTH_002",
					"message": "Authentication required",
				},
			})
		}

		// Get role from context
		role, ok := c.Locals(roleKey).(string)
		if !ok || role == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "AUTH_002",
					"message": "Role not found in token",
				},
			})
		}

		// Check if user has super_admin role
		if role != AdminRoleSuperAdmin {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "AUTH_004",
					"message": "Super admin access required",
				},
			})
		}

		return c.Next()
	}
}

// GetRole extracts role from context
func GetRole(c *fiber.Ctx) string {
	role, _ := c.Locals(roleKey).(string)
	return role
}
