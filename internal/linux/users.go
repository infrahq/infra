package linux

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/logging"
)

type LocalUser struct {
	Username string
	UID      string
	GID      string
	Info     []string
	HomeDir  string
}

const sentinelManagedByInfra = "managed by infra"

func (u LocalUser) IsManagedByInfra() bool {
	return len(u.Info) > 1 && u.Info[1] == sentinelManagedByInfra
}

// ReadLocalUsers reads a file in /etc/passwd format and returns the list of
// users in that file.
func ReadLocalUsers(filename string) ([]LocalUser, error) {
	fh, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer fh.Close() // read-only file, safe to ignore errors
	scan := bufio.NewScanner(fh)

	var result []LocalUser
	for scan.Scan() {
		line := strings.TrimSpace(scan.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, ":")
		if len(fields) < 7 {
			return nil, fmt.Errorf("invalid line contains less than 7 fields")
		}
		result = append(result, LocalUser{
			Username: fields[0],
			// field 1 is not used
			UID:     fields[2],
			GID:     fields[3],
			Info:    strings.FieldsFunc(fields[4], isRuneComma),
			HomeDir: fields[5],
			// field 6 is login shell
		})
	}
	return result, scan.Err()
}

func isRuneComma(r rune) bool {
	return r == ','
}

func AddUser(user *api.User, group string) error {
	args := []string{
		"--comment", fmt.Sprintf("%v,%v", user.ID, sentinelManagedByInfra),
		"-m", "-p", "*", "-g", group, user.SSHLoginName,
	}
	cmd := exec.Command("useradd", args...)
	cmd.Stdout = logging.L
	cmd.Stderr = logging.L
	return cmd.Run()
}

func KillUserProcesses(user LocalUser) error {
	//nolint:gosec
	cmd := exec.Command("pkill", "--signal", "KILL", "--uid", user.Username)
	cmd.Stdout = logging.L
	cmd.Stderr = logging.L
	err := cmd.Run()

	var exitError *exec.ExitError
	switch {
	// if no processes are running, pkill exits with 1
	case errors.As(err, &exitError) && exitError.ExitCode() == 1:
		return nil
	case err != nil:
		return fmt.Errorf("kill processes: %w", err)
	}
	return nil
}

func RemoveUser(user LocalUser) error {
	//nolint:gosec
	cmd := exec.Command("userdel", "--remove", user.Username)
	cmd.Stdout = logging.L
	cmd.Stderr = logging.L
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("userdel: %w", err)
	}
	return nil
}
