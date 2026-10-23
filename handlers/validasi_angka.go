package handlers

import (
	"regexp"
	"strings"
)

var digitsOnlyRegex = regexp.MustCompile(`^[0-9]+$`)

// validNumericOptional: nil atau kosong dianggap valid, selain itu harus angka saja.
func validNumericOptional(value *string) bool {
	if value == nil {
		return true
	}
	v := strings.TrimSpace(*value)
	return v == "" || digitsOnlyRegex.MatchString(v)
}
