package secrets

import (
	"context"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

func waitForSecretsManagerReady(t *testing.T, ssm *secretsmanager.SecretsManager) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	for {
		// nolint
		resp, err := http.Get(ssm.Client.Endpoint)
		// server responds with 404 and body of status running 😂
		if err == nil {
			b, err := ioutil.ReadAll(resp.Body)
			resp.Body.Close()

			if err == nil {
				if strings.Contains(string(b), "running") {
					return // ready!
				}
			}
		}

		if ctx.Err() != nil {
			t.Error("timeout waiting for secrets manager to be ready")
			t.FailNow()

			return
		}

		time.Sleep(100 * time.Millisecond)
	}
}
