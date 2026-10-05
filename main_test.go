package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"

	"e-class-backend/config"
)

func apiRequest(t *testing.T, router http.Handler, method, path string, payload any, token string) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	if payload != nil {
		if err := json.NewEncoder(&body).Encode(payload); err != nil {
			t.Fatal(err)
		}
	}
	request := httptest.NewRequest(method, "/api"+path, &body)
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func decodeObject(t *testing.T, response *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var result map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON response: %v (%s)", err, response.Body.String())
	}
	return result
}

func assertStatus(t *testing.T, response *httptest.ResponseRecorder, expected int) {
	t.Helper()
	if response.Code != expected {
		t.Fatalf("expected HTTP %d, received %d: %s", expected, response.Code, response.Body.String())
	}
}

func assertListHasID(t *testing.T, response *httptest.ResponseRecorder, id int) {
	t.Helper()
	var rows []map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &rows); err != nil {
		t.Fatalf("invalid list response: %v", err)
	}
	for _, row := range rows {
		if int(row["id"].(float64)) == id {
			return
		}
	}
	t.Fatalf("list response did not contain id %d", id)
}

func TestAdminCRUDRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("JWT_SECRET", "crud-test-secret")
	t.Setenv("DATABASE_PATH", filepath.Join(t.TempDir(), "eclass-test.db"))
	config.ConnectDB()
	database, err := config.DB.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	seedAdmin()
	router := setupRouter()

	login := apiRequest(t, router, http.MethodPost, "/auth/login", map[string]any{"username": "admin", "password": "admin123"}, "")
	assertStatus(t, login, http.StatusOK)
	adminToken := decodeObject(t, login)["token"].(string)
	self := decodeObject(t, apiRequest(t, router, http.MethodGet, "/profile", nil, adminToken))
	selfID := int(self["id"].(float64))
	for _, name := range []string{"Admin Edit One", "Admin Edit Two"} {
		update := apiRequest(t, router, http.MethodPut, "/admin/"+jsonNumber(selfID), map[string]any{"nama": name, "email": "admin", "username": "admin", "role": "admin"}, adminToken)
		assertStatus(t, update, http.StatusOK)
	}

	accountRoutes := []struct{ role, path string }{
		{"admin", "/admin"}, {"admin_kurikulum", "/admin-kurikulum"}, {"kepala_sekolah", "/kepala-sekolah"},
	}
	accountIDs := make(map[string]int)
	for _, route := range accountRoutes {
		body := map[string]any{"nama": "Test " + route.role, "username": "crud_" + route.role, "email": "crud_" + route.role + "@example.test", "password": "crud-pass-123", "status": "aktif", "nip": "123456789"}
		created := apiRequest(t, router, http.MethodPost, route.path, body, adminToken)
		assertStatus(t, created, http.StatusCreated)
		id := int(decodeObject(t, created)["id"].(float64))
		accountIDs[route.path] = id

		list := apiRequest(t, router, http.MethodGet, route.path, nil, adminToken)
		assertStatus(t, list, http.StatusOK)
		assertListHasID(t, list, id)
		assertStatus(t, apiRequest(t, router, http.MethodGet, route.path+"/"+jsonNumber(id), nil, adminToken), http.StatusOK)
		body["nama"] = "Updated " + route.role
		assertStatus(t, apiRequest(t, router, http.MethodPut, route.path+"/"+jsonNumber(id), body, adminToken), http.StatusOK)
		updated := decodeObject(t, apiRequest(t, router, http.MethodGet, route.path+"/"+jsonNumber(id), nil, adminToken))
		if updated["nama"] != body["nama"] {
			t.Fatalf("%s update was not returned by detail", route.path)
		}
	}

	major := apiRequest(t, router, http.MethodPost, "/jurusan", map[string]any{"nama": "Test Major", "kode": "T-MAJOR", "kepala_jurusan": "Initial"}, adminToken)
	assertStatus(t, major, http.StatusCreated)
	majorID := int(decodeObject(t, major)["id"].(float64))
	assertStatus(t, apiRequest(t, router, http.MethodGet, "/jurusan", nil, adminToken), http.StatusOK)
	assertStatus(t, apiRequest(t, router, http.MethodGet, "/jurusan/"+jsonNumber(majorID), nil, adminToken), http.StatusOK)
	assertStatus(t, apiRequest(t, router, http.MethodPut, "/jurusan/"+jsonNumber(majorID), map[string]any{"nama": "Updated Major"}, adminToken), http.StatusOK)

	class := apiRequest(t, router, http.MethodPost, "/kelas", map[string]any{"nama": "X Test 1", "tingkat": "X", "jurusan_id": majorID}, adminToken)
	assertStatus(t, class, http.StatusCreated)
	classID := int(decodeObject(t, class)["id"].(float64))
	classList := apiRequest(t, router, http.MethodGet, "/kelas?jurusan_id="+jsonNumber(majorID), nil, adminToken)
	assertStatus(t, classList, http.StatusOK)
	assertListHasID(t, classList, classID)
	assertStatus(t, apiRequest(t, router, http.MethodGet, "/kelas/"+jsonNumber(classID), nil, adminToken), http.StatusOK)
	assertStatus(t, apiRequest(t, router, http.MethodPut, "/kelas/"+jsonNumber(classID), map[string]any{"nama": "XI Test 1", "tingkat": "XI"}, adminToken), http.StatusOK)

	subject := apiRequest(t, router, http.MethodPost, "/pelajaran", map[string]any{"nama": "Test Subject", "kode": "T-SUB"}, adminToken)
	assertStatus(t, subject, http.StatusCreated)
	subjectID := int(decodeObject(t, subject)["id"].(float64))
	if len(decodeObject(t, subject)["guru_ids"].([]any)) != 0 {
		t.Fatal("subject create must allow an empty teacher list")
	}
	assertStatus(t, apiRequest(t, router, http.MethodGet, "/pelajaran", nil, adminToken), http.StatusOK)
	assertStatus(t, apiRequest(t, router, http.MethodGet, "/pelajaran/"+jsonNumber(subjectID), nil, adminToken), http.StatusOK)
	assertStatus(t, apiRequest(t, router, http.MethodPost, "/pelajaran", map[string]any{"nama": "Duplicate", "kode": "t-sub"}, adminToken), http.StatusConflict)
	assertStatus(t, apiRequest(t, router, http.MethodPut, "/pelajaran/"+jsonNumber(subjectID), map[string]any{"nama": "Updated Subject", "kode": "T-SUB-2"}, adminToken), http.StatusOK)

	teacher := apiRequest(t, router, http.MethodPost, "/guru", map[string]any{"nama": "Test Teacher", "username": "test_teacher", "email": "teacher@example.test", "password": "crud-pass-123", "status": "aktif", "nip": "123456789"}, adminToken)
	assertStatus(t, teacher, http.StatusCreated)
	teacherID := int(decodeObject(t, teacher)["id"].(float64))
	teacherList := apiRequest(t, router, http.MethodGet, "/guru?search=Test+Teacher", nil, adminToken)
	assertStatus(t, teacherList, http.StatusOK)
	assertListHasID(t, teacherList, teacherID)
	assertStatus(t, apiRequest(t, router, http.MethodGet, "/guru/"+jsonNumber(teacherID), nil, adminToken), http.StatusOK)
	assertStatus(t, apiRequest(t, router, http.MethodPut, "/pelajaran/"+jsonNumber(subjectID), map[string]any{"guru_ids": []int{teacherID}}, adminToken), http.StatusOK)
	assertStatus(t, apiRequest(t, router, http.MethodGet, "/pelajaran/"+jsonNumber(subjectID)+"/guru", nil, adminToken), http.StatusOK)
	assertStatus(t, apiRequest(t, router, http.MethodPut, "/guru/"+jsonNumber(teacherID), map[string]any{"nama": "Updated Teacher", "nip": "987654321"}, adminToken), http.StatusOK)
	assertStatus(t, apiRequest(t, router, http.MethodPut, "/guru/"+jsonNumber(teacherID), map[string]any{"nama": "Updated Teacher Again", "nip": "987654322"}, adminToken), http.StatusOK)

	student := apiRequest(t, router, http.MethodPost, "/siswa", map[string]any{"nama": "Test Student", "username": "test_student", "email": "student@example.test", "password": "crud-pass-123", "status": "aktif", "nisn": "1234567890", "kelas_id": classID}, adminToken)
	assertStatus(t, student, http.StatusCreated)
	studentID := int(decodeObject(t, student)["id"].(float64))
	assertStatus(t, apiRequest(t, router, http.MethodGet, "/siswa?jurusan_id="+jsonNumber(majorID)+"&kelas_id="+jsonNumber(classID)+"&search=Test+Student", nil, adminToken), http.StatusOK)
	assertStatus(t, apiRequest(t, router, http.MethodGet, "/siswa/"+jsonNumber(studentID), nil, adminToken), http.StatusOK)
	assertStatus(t, apiRequest(t, router, http.MethodPut, "/siswa/"+jsonNumber(studentID), map[string]any{"nama": "Updated Student", "status": "nonaktif"}, adminToken), http.StatusOK)
	assertStatus(t, apiRequest(t, router, http.MethodPut, "/siswa/"+jsonNumber(studentID), map[string]any{"nisn": "1234567891"}, adminToken), http.StatusOK)
	assertStatus(t, apiRequest(t, router, http.MethodPut, "/siswa/"+jsonNumber(studentID), map[string]any{"nama": "Updated Student Again"}, adminToken), http.StatusOK)
	assertStatus(t, apiRequest(t, router, http.MethodPut, "/siswa/"+jsonNumber(studentID), map[string]any{"nisn": "123ABC"}, adminToken), http.StatusBadRequest)
	unassigned := apiRequest(t, router, http.MethodPut, "/siswa/"+jsonNumber(studentID), map[string]any{"kelas_id": nil}, adminToken)
	assertStatus(t, unassigned, http.StatusOK)
	if decodeObject(t, apiRequest(t, router, http.MethodGet, "/siswa/"+jsonNumber(studentID), nil, adminToken))["kelas_id"] != nil {
		t.Fatal("student class assignment was not cleared")
	}
	assertStatus(t, apiRequest(t, router, http.MethodPut, "/siswa/"+jsonNumber(studentID), map[string]any{"kelas_id": classID}, adminToken), http.StatusOK)
	studentList := apiRequest(t, router, http.MethodGet, "/siswa?jurusan_id="+jsonNumber(majorID)+"&kelas_id="+jsonNumber(classID)+"&status=nonaktif&search=Updated+Student", nil, adminToken)
	assertStatus(t, studentList, http.StatusOK)
	assertListHasID(t, studentList, studentID)
	noStudents := apiRequest(t, router, http.MethodGet, "/siswa?search=does-not-exist", nil, adminToken)
	assertStatus(t, noStudents, http.StatusOK)
	if len(bytes.TrimSpace(noStudents.Body.Bytes())) > 2 {
		t.Fatal("nonmatching student search returned rows")
	}

	kurikulumToken := loginToken(t, router, "crud_admin_kurikulum", "crud-pass-123")
	assertStatus(t, apiRequest(t, router, http.MethodPost, "/pelajaran", map[string]any{"nama": "Denied", "kode": "DENIED"}, kurikulumToken), http.StatusForbidden)
	studentToken := loginToken(t, router, "test_student", "crud-pass-123")
	for _, path := range []string{"/jurusan", "/kelas", "/pelajaran"} {
		assertStatus(t, apiRequest(t, router, http.MethodGet, path, nil, studentToken), http.StatusForbidden)
	}

	assertStatus(t, apiRequest(t, router, http.MethodDelete, "/siswa/"+jsonNumber(studentID), nil, adminToken), http.StatusOK)
	assertStatus(t, apiRequest(t, router, http.MethodDelete, "/guru/"+jsonNumber(teacherID), nil, adminToken), http.StatusOK)
	assertStatus(t, apiRequest(t, router, http.MethodDelete, "/pelajaran/"+jsonNumber(subjectID), nil, adminToken), http.StatusOK)
	assertStatus(t, apiRequest(t, router, http.MethodDelete, "/kelas/"+jsonNumber(classID), nil, adminToken), http.StatusOK)
	assertStatus(t, apiRequest(t, router, http.MethodDelete, "/jurusan/"+jsonNumber(majorID), nil, adminToken), http.StatusOK)
	for _, route := range accountRoutes {
		id := accountIDs[route.path]
		assertStatus(t, apiRequest(t, router, http.MethodDelete, route.path+"/"+jsonNumber(id), nil, adminToken), http.StatusOK)
		assertStatus(t, apiRequest(t, router, http.MethodGet, route.path+"/"+jsonNumber(id), nil, adminToken), http.StatusNotFound)
	}
}

func loginToken(t *testing.T, router http.Handler, username, password string) string {
	t.Helper()
	response := apiRequest(t, router, http.MethodPost, "/auth/login", map[string]any{"username": username, "password": password}, "")
	assertStatus(t, response, http.StatusOK)
	return decodeObject(t, response)["token"].(string)
}

func jsonNumber(value int) string {
	return strconv.Itoa(value)
}

func TestProfileRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("JWT_SECRET", "profile-test-secret")
	t.Setenv("DATABASE_PATH", filepath.Join(t.TempDir(), "profile-test.db"))
	config.ConnectDB()
	database, err := config.DB.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	seedAdmin()
	router := setupRouter()
	token := loginToken(t, router, "admin", "admin123")
	assertStatus(t, apiRequest(t, router, http.MethodGet, "/profile", nil, token), http.StatusOK)
	updated := apiRequest(t, router, http.MethodPut, "/profile", map[string]any{"nama": "Admin Updated", "email": "admin@example.test", "username": "admin", "telepon": "", "alamat": "", "foto_url": ""}, token)
	assertStatus(t, updated, http.StatusOK)
	if decodeObject(t, updated)["nama"] != "Admin Updated" {
		t.Fatal("profile update did not return saved name")
	}
	assertStatus(t, apiRequest(t, router, http.MethodPut, "/profile/password", map[string]any{"password_lama": "wrong-password", "password_baru": "new-password-123"}, token), http.StatusBadRequest)
	assertStatus(t, apiRequest(t, router, http.MethodPut, "/profile/password", map[string]any{"password_lama": "admin123", "password_baru": "new-password-123"}, token), http.StatusOK)
	assertStatus(t, apiRequest(t, router, http.MethodPost, "/auth/login", map[string]any{"username": "admin", "password": "new-password-123"}, ""), http.StatusOK)
	newToken := loginToken(t, router, "admin", "new-password-123")
	assertStatus(t, apiRequest(t, router, http.MethodPut, "/profile/password", map[string]any{"password_lama": "new-password-123", "password_baru": "admin123"}, newToken), http.StatusOK)
}
