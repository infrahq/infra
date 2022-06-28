package logging

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/rs/zerolog"
)

var strInvalidHeaderFieldValue = []byte("invalid header field value")

type FilteredHTTPLogger struct {
	zerolog.Logger
}

func (l FilteredHTTPLogger) Write(b []byte) (int, error) {
	if idx := bytes.Index(b, strInvalidHeaderFieldValue); idx >= 0 {
		idx += len(strInvalidHeaderFieldValue)

		forKeyIdx := bytes.Index(b, []byte("for key"))
		if forKeyIdx > idx {
			return l.Logger.Write(append(b[:idx+1], b[forKeyIdx:]...))
		}

		if b[0] != '{' {
			// not json; free to truncate.
			return l.Logger.Write(b[:idx])
		}

		// we can't see where the end is. parse the message so you can truncate it. :/
		m := map[string]interface{}{}
		if err := json.Unmarshal(b, &m); err != nil {
			Errorf("Had some trouble parsing log line that needs to be filtered. Omitting log entry")
			// on error write nothing, just to be safe.
			return 0, nil // nolint
		}

		if msg, ok := m["msg"]; ok {
			if smsg, ok := msg.(string); ok {
				if idx := strings.Index(smsg, string(strInvalidHeaderFieldValue)); idx >= 0 {
					m["msg"] = smsg[:idx+len(strInvalidHeaderFieldValue)]

					newBytes, err := json.Marshal(m)
					if err == nil {
						return l.Logger.Write(newBytes)
					}
				}
			}
		}

		// write nothing, just to be safe.
		return 0, nil
	}

	return l.Logger.Write(b)
}

func NewFilteredHTTPLogger() *FilteredHTTPLogger {
	return &FilteredHTTPLogger{
		L.With().Logger(),
	}
}
