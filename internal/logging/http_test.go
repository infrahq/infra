package logging

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/rs/zerolog"
	"gotest.tools/v3/assert"
)

func TestFiltersOutBearerTokenValue(t *testing.T) {
	// remove this test after Go patches https://github.com/golang/go/pull/48979
	tests := []struct {
		Input    string
		Expected string
	}{
		{
			Input:    `could not create destination: Post \"https://{anonymized}\": net/http: invalid header field value \"Bearer {anonymized_token}\\n\" for key Authorization`,
			Expected: `could not create destination: Post \"https://{anonymized}\": net/http: invalid header field value for key Authorization`,
		},
		{
			Input:    `could not create destination: Post \"https://{anonymized}\": net/http: invalid header field value \"Bearer {anonymized_token}\\n\"`,
			Expected: `could not create destination: Post \"https://{anonymized}\": net/http: invalid header field value`,
		},
		{
			Input:    `{"level":"error","ts":1637161597.990963,"caller":"connector/connector.go:400","msg":"could not create destination: Post \"https://{anonymized}\": net/http: invalid header field value \"Bearer {anonymized_token}\\n\" for key Authorization"}`,
			Expected: `{"level":"error","ts":1637161597.990963,"caller":"connector/connector.go:400","msg":"could not create destination: Post \"https://{anonymized}\": net/http: invalid header field value for key Authorization"}`,
		},
	}
	for _, testCase := range tests {
		t.Run("", func(t *testing.T) {
			b := &bytes.Buffer{}
			logger := FilteredHTTPLogger{zerolog.New(b)}
			n, err := logger.Write([]byte(testCase.Input))
			assert.NilError(t, err)
			assert.Equal(t, n, len(testCase.Expected))

			m := map[string]interface{}{}
			err = json.Unmarshal(b.Bytes(), &m)
			assert.NilError(t, err, b.String())

			msg, ok := m["message"].(string)
			assert.Assert(t, ok)
			assert.Equal(t, testCase.Expected, msg)
		})
	}
}
