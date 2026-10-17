package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Role pemantau (admin_kurikulum, kepala_sekolah) hanya boleh membaca data.
func isReadOnlyRole(role string) bool {
	return role == "admin_kurikulum" || role == "kepala_sekolah"
}

func isSafeMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions
}

// blockReadOnlyWrite menolak (403) setiap request tulis dari role pemantau.
// Dipanggil dari VerifyToken sehingga berlaku untuk semua route yang butuh login.
func blockReadOnlyWrite(c *gin.Context) bool {
	if isReadOnlyRole(c.GetString("role")) && !isSafeMethod(c.Request.Method) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Role ini hanya dapat melihat dan memantau data"})
		return true
	}
	return false
}
