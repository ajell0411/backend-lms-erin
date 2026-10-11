package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"e-class-backend/models"
)

func guruPortalToUint(v interface{}) (uint, bool) {
	switch n := v.(type) {
	case uint:
		return n, n > 0
	case uint32:
		return uint(n), n > 0
	case uint64:
		return uint(n), n > 0
	case int:
		return uint(n), n > 0
	case int64:
		return uint(n), n > 0
	case float64:
		return uint(n), n > 0
	case string:
		x, err := strconv.ParseUint(n, 10, 64)
		return uint(x), err == nil && x > 0
	}
	return 0, false
}

// guruPortalID mengambil ID guru yang sedang login dari context (diisi middleware.VerifyToken).
func guruPortalID(c *gin.Context) (uint, bool) {
	for _, key := range []string{"userID", "user_id", "userId", "id", "uid"} {
		if v, ok := c.Get(key); ok {
			if id, ok := guruPortalToUint(v); ok {
				return id, true
			}
		}
	}
	if v, ok := c.Get("user"); ok {
		switch u := v.(type) {
		case models.User:
			return u.ID, true
		case *models.User:
			if u != nil {
				return u.ID, true
			}
		}
	}
	c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Sesi tidak valid, silakan login lagi"})
	return 0, false
}

func guruPortalParamID(c *gin.Context) (uint, bool) {
	n, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || n == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "ID tidak valid"})
		return 0, false
	}
	return uint(n), true
}

func guruPortalGagal(c *gin.Context, status int, pesan string) {
	c.JSON(status, gin.H{"success": false, "message": pesan})
}

func guruPortalString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
