package email

import (
	"fmt"
	"strings"
)

func Domain(email string) (string, error) {
	at := strings.LastIndex(email, "@") // get the last @ since the email spec allows for multiple @s
	if at == -1 {
		return "", fmt.Errorf("%s is an invalid email address", email)
	}
	return email[at+1:], nil
}
