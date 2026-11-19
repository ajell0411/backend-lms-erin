package handlers

import (
	"testing"

	"e-class-backend/models"
)

func TestStudentProfileIdentityCannotBeChanged(t *testing.T) {
	username := "siswa01"
	user := models.User{Nama: "Siswa Contoh", Username: &username}
	changedUsername := "usernamebaru"
	unchangedUsername := "siswa01"

	tests := []struct {
		name  string
		input profileInput
		want  bool
	}{
		{name: "other profile fields only", input: profileInput{Email: "siswa@example.com"}, want: true},
		{name: "same identity", input: profileInput{Nama: "Siswa Contoh", Username: &unchangedUsername}, want: true},
		{name: "changed name", input: profileInput{Nama: "Nama Baru"}, want: false},
		{name: "changed username", input: profileInput{Username: &changedUsername}, want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := studentProfileIdentityUnchanged(user, test.input); got != test.want {
				t.Fatalf("studentProfileIdentityUnchanged() = %v, want %v", got, test.want)
			}
		})
	}
}
