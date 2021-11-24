package logging

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestConfigDefault(t *testing.T) {
	logger, _ := Initialize(0)

	assert.NotNil(t, logger)

	if checked := logger.Check(zap.InfoLevel, "default"); checked == nil {
		assert.Fail(t, "could not log info level messages")
	}

	if checked := logger.Check(zap.DebugLevel, "not default"); checked != nil {
		assert.Fail(t, "should not log debug level messages")
	}
}

func TestConfigValidLevel(t *testing.T) {
	logger, _ := Initialize(1)

	assert.NotNil(t, logger)

	if checked := logger.Check(zap.DebugLevel, "not default"); checked == nil {
		assert.Fail(t, "could not log debug level messages")
	}
}

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
			Input:    `{"level":"error","ts":1637161597.990963,"caller":"engine/engine.go:400","msg":"could not create destination: Post \"https://{anonymized}\": net/http: invalid header field value \"Bearer {anonymized_token}\\n\" for key Authorization"}`,
			Expected: `{"level":"error","ts":1637161597.990963,"caller":"engine/engine.go:400","msg":"could not create destination: Post \"https://{anonymized}\": net/http: invalid header field value for key Authorization"}`,
		},
	}
	for _, testCase := range tests {
		writeSyncer := &testWriterSyncer{}
		defaultStdoutWriter = writeSyncer
		defaultStderrWriter = writeSyncer

		logger, err := Initialize(int(zap.InfoLevel))
		require.NoError(t, err)

		logger.Sugar().Info(testCase.Input)

		m := map[string]interface{}{}
		err = json.Unmarshal(writeSyncer.data, &m)
		require.NoError(t, err)

		require.Equal(t, testCase.Expected, m["msg"].(string))
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
