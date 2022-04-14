package kubernetes

import (
	"testing"

	"github.com/go-playground/validator/v10"
	"gotest.tools/assert"
)

func TestFQDN(t *testing.T) {
	type Testo struct {
		URL string `validate:"fqdn"`
	}

	teste := Testo{URL: "test:1234"}
	err := validator.New().Struct(teste)
	assert.NilError(t, err)
}

// func TestName(t *testing.T) {
// 	// handler := func(resp http.ResponseWriter, req *http.Request) {
// 	// 	// if req.URL.Path != "/v1/logout" {
// 	// 	// 	resp.WriteHeader(http.StatusBadRequest)
// 	// 	// 	return
// 	// 	// }
// 	// 	resp.WriteHeader(http.StatusOK)
// 	// 	_, _ = resp.Write([]byte(`{}`)) // API client requires a JSON response
// 	// }

// 	// srv := httptest.NewTLSServer(http.HandlerFunc(handler))
// 	// t.Cleanup(srv.Close)

// 	k := &Kubernetes{
// 		Config: ,
// 	}
// 	_, err := k.Name("12345678910112")
// 	assert.NilError(t, err)
// }
