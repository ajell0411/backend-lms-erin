package handlers

import "regexp"

var passwordRe = regexp.MustCompile(`^[A-Za-z0-9]{8,}$`)

// validPassword: minimal 8 karakter, hanya huruf dan angka.
func validPassword(p string) bool { return passwordRe.MatchString(p) }
