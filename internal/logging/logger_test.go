package logging

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
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
		writeSyncer := &testWriterSyncer{}
		defaultStdoutWriter = writeSyncer
		defaultStderrWriter = writeSyncer

		logger, err := NewLogger(zapcore.InfoLevel)
		require.NoError(t, err)

		logger.Sugar().Info(testCase.Input)

		m := map[string]interface{}{}
		err = json.Unmarshal(writeSyncer.data, &m)
		require.NoError(t, err)

		msg, ok := m["msg"].(string)
		require.True(t, ok)
		require.Equal(t, testCase.Expected, msg)
	}
}

type testWriterSyncer struct {
	data []byte
}

func (w *testWriterSyncer) Write(b []byte) (int, error) {
	w.data = append(w.data, b...)
	return len(b), nil
}

func (*testWriterSyncer) Sync() error {
	return nil
}
