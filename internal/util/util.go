package util

import "crypto/rand"

func RandString(n int) string {
	if n < 0 {
		return RandString(0)
	}

	const alphanum = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	var bytes = make([]byte, n)
	rand.Read(bytes)
	for i, b := range bytes {
		bytes[i] = alphanum[b%byte(len(alphanum))]
	}

	return string(bytes)
}

func ValidPermission(permission string) bool {
	for _, p := range []string{"view", "edit", "admin"} {
		if p == permission {
			return true
		}
	}
	return false
}
