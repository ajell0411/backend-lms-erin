package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func tokenUntukRole(t *testing.T, role string) string {
	t.Helper()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userId": 1,
		"role":   role,
		"exp":    time.Now().Add(time.Hour).Unix(),
	}).SignedString([]byte("rahasia-test"))
	if err != nil {
		t.Fatalf("gagal membuat token: %v", err)
	}
	return token
}

func routerUji() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	ok := func(c *gin.Context) { c.Status(http.StatusOK) }

	g := r.Group("/umum", VerifyToken())
	g.GET("", ok)
	g.POST("", ok)
	g.PUT("", ok)
	g.PATCH("", ok)
	g.DELETE("", ok)

	a := r.Group("/admin", VerifyToken(), RequireAdminWrite())
	a.POST("", ok)
	a.PUT("", ok)
	a.DELETE("", ok)
	return r
}

func kirim(r *gin.Engine, method, path, token string) int {
	req := httptest.NewRequest(method, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code
}

func TestRolePemantauanTidakBisaMenulis(t *testing.T) {
	t.Setenv("JWT_SECRET", "rahasia-test")
	r := routerUji()

	for _, role := range []string{"admin_kurikulum", "kepala_sekolah"} {
		token := tokenUntukRole(t, role)

		if code := kirim(r, http.MethodGet, "/umum", token); code != http.StatusOK {
			t.Errorf("%s: GET seharusnya 200, dapat %d", role, code)
		}
		for _, m := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
			if code := kirim(r, m, "/umum", token); code != http.StatusForbidden {
				t.Errorf("%s: %s /umum seharusnya 403, dapat %d", role, m, code)
			}
		}
		for _, m := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
			if code := kirim(r, m, "/admin", token); code != http.StatusForbidden {
				t.Errorf("%s: %s /admin seharusnya 403, dapat %d", role, m, code)
			}
		}
	}
}

func TestRoleLainTidakTerganggu(t *testing.T) {
	t.Setenv("JWT_SECRET", "rahasia-test")
	r := routerUji()

	for _, role := range []string{"admin", "guru", "siswa"} {
		token := tokenUntukRole(t, role)
		for _, m := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete} {
			if code := kirim(r, m, "/umum", token); code != http.StatusOK {
				t.Errorf("%s: %s /umum seharusnya 200, dapat %d", role, m, code)
			}
		}
	}
	if code := kirim(r, http.MethodPost, "/admin", tokenUntukRole(t, "admin")); code != http.StatusOK {
		t.Errorf("admin: POST /admin seharusnya 200, dapat %d", code)
	}
	if code := kirim(r, http.MethodPost, "/umum", ""); code != http.StatusUnauthorized {
		t.Errorf("tanpa token seharusnya 401, dapat %d", code)
	}
}
