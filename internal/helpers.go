package internal

import (
	"math/rand"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	letterRunes = []rune("abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ2345679")
)

// RandStringRunes returns a random string with n characters
func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}

	// due to PostgreSQL schema requiring letter start.
	b[0] = letterRunes[0]

	return string(b)
}

// CleanUpFileName removes file extention and anything but a-zA-Z-_
func CleanUpFileName(s string) string {
	s = strings.TrimSuffix(s, filepath.Ext(s))

	exp := regexp.MustCompile(`[^a-zA-Z\-_]`)

	return exp.ReplaceAllString(s, "")

}
