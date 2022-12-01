package connector

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/api"
	data "github.com/infrahq/infra/internal/linux"
	"github.com/infrahq/infra/uid"
)

func TestUpdateLocalUsers(t *testing.T) {
	logDir := t.TempDir()
	logFile := filepath.Join(logDir, "users.log")
	cwd, _ := os.Getwd()
	t.Setenv("PATH", filepath.Join(cwd, "testdata/bin")+":"+os.Getenv("PATH"))
	t.Setenv("TEST_CONNECTOR_USER_LOG_FILE", logFile)

	etcPasswdFilename = "testdata/localusers-etcpasswd"
	t.Cleanup(func() {
		etcPasswdFilename = "/etc/passwd"
	})

	ctx := context.Background()
	fakeClient := &fakeAPIClient{
		users: map[uid.ID]api.User{
			1111: {ID: 1111, Name: "one@example.com", SSHLoginName: "one111"},
			2222: {ID: 2222, Name: "two@example.com", SSHLoginName: "two222"},
		},
	}
	grants := []api.Grant{
		{ID: 123, User: 1111, Privilege: "connect"},
		{ID: 124, User: 2222, Privilege: "connect"},
	}

	err := updateLocalUsers(ctx, fakeClient, grants)
	assert.NilError(t, err)

	actual, err := os.ReadFile(logFile)
	assert.NilError(t, err)

	expected := `pkill '--uid' 'three333' 
userdel '--remove' 'three333' 
useradd '--comment' 'Ej,managed by infra' '-m' '-p' '*' 'two222' 
`
	assert.Equal(t, expected, string(actual))
}

func TestReadHostKeysFromDir(t *testing.T) {
	hostKeys, err := readHostKeysFromDir("./testdata/etcssh")
	assert.NilError(t, err)

	var expected = `ssh-dss AAAAB3NzaC1kc3MAAACBAIw6DfiYR9DDi/iojqjhM0mlhZ6K+QMukZv2S/Su/M4QmpPhLMgvz16QCS2Wo6y4No6XdTKqp8/RCRobQA6rELoNZHc4IDylwuu7/xn1tdLF5vxUgiz9YMFQDm8rbltA0Gpc6CaKmu0OIJmHKCUZNWoteXa+d9CaYNsc8DL7T3ChAAAAFQDpnbY6PEDh6plGF9hK1eiGPh1WuwAAAIAiAH3Ig+BfgQxk6fLzuTmZDxCAAfWy3dT28eH1Bhef+W5/kVU4zPg4MUnhruQM+ViGrjRyoMQyJJeVBnOkvVvBg5QZi5rMP5OkW5VZ3hTA7tMbouPiOFVUBUlNW4wl/tDr8BpGUlHU0MuO1o4hTV2lDPwjfNV1nX+um+Zdek0DggAAAIByYDA/oVFig/zPFcYGL3NgJ5Dttr3O0uJ6pcopZVzarYIBvWmqDdrHnK12H5SEnCgC4a3g9xtmD0F+A4va/M6jLD/aWbqmgy7g9zAflTH2NroA0UlmcbZqwMM8ITdPWpxOHQPmGuBwhQ4fZ4CTjtTzzUsWvcxlGOgU0KXHtCZi0g==
ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBI928HrNO4w5wneti+mBYBC1ZGr0oVxUJTr5AwXoN2YEYZj/T+LSFEPrLRzyWBycpyPwld0AVhr1hm3mG9U6N/w=
ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIJluZNrFxN0dfBrJW4rebQnTjwFxP+WLoN1QnbjRoVvZ
ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDNpxkQM9F1eTnfcxjLSqJ1W5v+e8x2jKpHPZdpB9Ug4OMD/F4lQi1Q7Ut+8L6Mnu1AlsahQGMSNCibITlTM+6Zk0UHpFZdaGz7v6lrMwcLP2bfcrvpbONZI1D0CG16VFTW3Gd7v5AdaqnaM4mog1+4C6K1SEwytX0YT1BBdBknIoM8thgZCYkZgJgnNoXasr0mN86LTCOfM2w6vIp3f2Zvlc8zmv4IFZ742mZXCLH1H0A1/0mRsj5iGJeRUPzQVHz6rxlz8kiaCDIhXRCSirR/TRiahxwgOHbrgv7kgugsdBapkmBBX5OsWakYFz+UCn9Fwf9uq/enVhKNn56Nq94p0WXTOl2nMKzVJb0gpBaWn14uH14/VOPxhzUh3GprW2FXJp5DXI2q8GrMK4EaHCkzm1gU5WVWbVPBQmfdpW6kCMlftsyDNACB5mZEmnQcT6mxH+xgQn+irLp34SpnUODkyhm9bV5hSDrhrzgRd3D8NVXf/YiZFWHRaa49kddrm+8=
`
	assert.DeepEqual(t, hostKeys, expected)
}

func TestReadLocalUsers(t *testing.T) {
	actual, err := data.ReadLocalUsers("./testdata/etcpasswd")
	assert.NilError(t, err)
	expected := []data.LocalUser{
		{
			Username: "root",
			Info:     []string{"root"},
			UID:      "0",
			GID:      "0",
			HomeDir:  "/root",
		},
		{
			Username: "adm",
			Info:     []string{},
			UID:      "3",
			GID:      "4",
			HomeDir:  "/var/adm",
		},
		{
			Username: "example",
			Info:     []string{"example", "managed by infra"},
			UID:      "1001",
			GID:      "1001",
			HomeDir:  "/home/example",
		},
	}
	assert.DeepEqual(t, actual, expected)
	assert.Assert(t, actual[2].IsManagedByInfra())
}
