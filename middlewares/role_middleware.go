package middlewares

import (
	"gin-fleamarket/models"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// RoleBasedAccessControl 指定されたロールのみアクセスを許可するミドルウェア
// AuthMiddlewareの後に使用することを想定（ctxに"user"が設定されている必要がある）
func RoleBasedAccessControl(allowedRoles ...string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		user, exists := ctx.Get("user")
		if !exists {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		userModel, ok := user.(*models.User)
		if !ok {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// デバッグ用ログ
		log.Printf("RoleBasedAccessControl: User ID=%d, Email=%s, Role=%s, AllowedRoles=%v",
			userModel.ID, userModel.Email, userModel.Role, allowedRoles)

		// 許可されたロールかチェック（大文字小文字を無視、空白をトリム）
		hasAccess := false
		userRole := strings.TrimSpace(strings.ToLower(userModel.Role))
		for _, allowedRole := range allowedRoles {
			if userRole == strings.TrimSpace(strings.ToLower(allowedRole)) {
				hasAccess = true
				break
			}
		}

		if !hasAccess {
			log.Printf("RoleBasedAccessControl: Access denied. User role=%s, Required roles=%v",
				userModel.Role, allowedRoles)
			ctx.AbortWithStatus(http.StatusForbidden)
			return
		}

		ctx.Next()
	}
}
