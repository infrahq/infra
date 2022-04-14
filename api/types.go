package api

import (
	"strings"
	"time"

	"github.com/andanhm/go-prettytime"

	"github.com/infrahq/infra/uid"
)

type Resource struct {
	ID uid.ID `uri:"id" validate:"required"`
}

type Time time.Time

func (t Time) MarshalJSON() ([]byte, error) {
	if time.Time(t).IsZero() {
		return []byte("null"), nil
	}
	s := time.Time(t).UTC().Format(time.RFC3339)
	return []byte(`"` + s + `"`), nil
}

func (t *Time) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	if string(data) == `""` {
		return nil
	}
	s := strings.Trim(string(data), `"`)
	tmp, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return err
	}
	*t = Time(tmp.UTC())
	return nil
}

func (t Time) String() string {
	return time.Time(t).Format(time.RFC3339)
}

func (t Time) Format(layout string) string {
	return time.Time(t).Format(layout)
}

func (t Time) Equal(other Time) bool {
	return time.Time(t).Equal(time.Time(other))
}

func (t Time) Relative(zeroName ...string) string {
	time := time.Time(t)

	if time.IsZero() {
		if len(zeroName) == 0 {
			return "none"
		}
		return zeroName[0]
	}

	return prettytime.Format(time)
}

type Duration time.Duration

func (d Duration) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Duration(d).String() + `"`), nil
}

func (d *Duration) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	dur, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = Duration(dur)
	return nil
}

func (d Duration) String() string {
	return time.Duration(d).String()
}
