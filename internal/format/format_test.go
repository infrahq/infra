package format

import (
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func TestHumanDuration(t *testing.T) {
	day := 24 * time.Hour
	week := 7 * day
	month := 30 * day
	year := 365 * day

	assert.Equal(t, "Less than a second", HumanDuration(450*time.Millisecond))
	assert.Equal(t, "1 second", HumanDuration(1*time.Second))
	assert.Equal(t, "45 seconds", HumanDuration(45*time.Second))
	assert.Equal(t, "46 seconds", HumanDuration(46*time.Second))
	assert.Equal(t, "59 seconds", HumanDuration(59*time.Second))
	assert.Equal(t, "About a minute", HumanDuration(60*time.Second))
	assert.Equal(t, "About a minute", HumanDuration(1*time.Minute))
	assert.Equal(t, "3 minutes", HumanDuration(3*time.Minute))
	assert.Equal(t, "35 minutes", HumanDuration(35*time.Minute))
	assert.Equal(t, "35 minutes", HumanDuration(35*time.Minute+40*time.Second))
	assert.Equal(t, "45 minutes", HumanDuration(45*time.Minute))
	assert.Equal(t, "45 minutes", HumanDuration(45*time.Minute+40*time.Second))
	assert.Equal(t, "46 minutes", HumanDuration(46*time.Minute))
	assert.Equal(t, "59 minutes", HumanDuration(59*time.Minute))
	assert.Equal(t, "About an hour", HumanDuration(1*time.Hour))
	assert.Equal(t, "About an hour", HumanDuration(1*time.Hour+29*time.Minute))
	assert.Equal(t, "2 hours", HumanDuration(1*time.Hour+31*time.Minute))
	assert.Equal(t, "2 hours", HumanDuration(1*time.Hour+59*time.Minute))
	assert.Equal(t, "3 hours", HumanDuration(3*time.Hour))
	assert.Equal(t, "3 hours", HumanDuration(3*time.Hour+29*time.Minute))
	assert.Equal(t, "4 hours", HumanDuration(3*time.Hour+31*time.Minute))
	assert.Equal(t, "4 hours", HumanDuration(3*time.Hour+59*time.Minute))
	assert.Equal(t, "4 hours", HumanDuration(3*time.Hour+60*time.Minute))
	assert.Equal(t, "24 hours", HumanDuration(24*time.Hour))
	assert.Equal(t, "36 hours", HumanDuration(1*day+12*time.Hour))
	assert.Equal(t, "2 days", HumanDuration(2*day))
	assert.Equal(t, "7 days", HumanDuration(7*day))
	assert.Equal(t, "13 days", HumanDuration(13*day+5*time.Hour))
	assert.Equal(t, "2 weeks", HumanDuration(2*week))
	assert.Equal(t, "2 weeks", HumanDuration(2*week+4*day))
	assert.Equal(t, "3 weeks", HumanDuration(3*week))
	assert.Equal(t, "4 weeks", HumanDuration(4*week))
	assert.Equal(t, "4 weeks", HumanDuration(4*week+3*day))
	assert.Equal(t, "4 weeks", HumanDuration(1*month))
	assert.Equal(t, "6 weeks", HumanDuration(1*month+2*week))
	assert.Equal(t, "2 months", HumanDuration(2*month))
	assert.Equal(t, "2 months", HumanDuration(2*month+2*week))
	assert.Equal(t, "3 months", HumanDuration(3*month))
	assert.Equal(t, "3 months", HumanDuration(3*month+1*week))
	assert.Equal(t, "5 months", HumanDuration(5*month+2*week))
	assert.Equal(t, "13 months", HumanDuration(13*month))
	assert.Equal(t, "23 months", HumanDuration(23*month))
	assert.Equal(t, "24 months", HumanDuration(24*month))
	assert.Equal(t, "2 years", HumanDuration(24*month+2*week))
	assert.Equal(t, "3 years", HumanDuration(3*year+2*month))
}

func TestHumanTime(t *testing.T) {
	now := time.Now()

	t.Run("zero value", func(t *testing.T) {
		assert.Equal(t, HumanTime(time.Time{}, "never"), "never")
	})
	t.Run("time in the future", func(t *testing.T) {
		v := now.Add(48 * time.Hour)
		assert.Equal(t, HumanTime(v, ""), "2 days from now")
	})
	t.Run("time in the past", func(t *testing.T) {
		v := now.Add(-48 * time.Hour)
		assert.Equal(t, HumanTime(v, ""), "2 days ago")
	})
}

func TestExactDuration(t *testing.T) {
	assert.Equal(t, "1 millisecond", ExactDuration(1*time.Millisecond))
	assert.Equal(t, "10 milliseconds", ExactDuration(10*time.Millisecond))
	assert.Equal(t, "1 second", ExactDuration(1*time.Second))
	assert.Equal(t, "10 seconds", ExactDuration(10*time.Second))
	assert.Equal(t, "1 minute", ExactDuration(1*time.Minute))
	assert.Equal(t, "10 minutes", ExactDuration(10*time.Minute))
	assert.Equal(t, "1 hour", ExactDuration(1*time.Hour))
	assert.Equal(t, "10 hours", ExactDuration(10*time.Hour))
	assert.Equal(t, "1 hour 1 second", ExactDuration(1*time.Hour+1*time.Second))
	assert.Equal(t, "1 hour 10 seconds", ExactDuration(1*time.Hour+10*time.Second))
	assert.Equal(t, "1 hour 1 minute", ExactDuration(1*time.Hour+1*time.Minute))
	assert.Equal(t, "1 hour 10 minutes", ExactDuration(1*time.Hour+10*time.Minute))
	assert.Equal(t, "1 hour 1 minute 1 second", ExactDuration(1*time.Hour+1*time.Minute+1*time.Second))
	assert.Equal(t, "10 hours 10 minutes 10 seconds", ExactDuration(10*time.Hour+10*time.Minute+10*time.Second))
}
