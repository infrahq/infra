package database

import (
	"fmt"
	"os"
)

type MainT struct {
	cleanup []func()
}

func (t *MainT) Cleanup(f func()) {
	// prepend so that they run in the correct order
	t.cleanup = append([]func(){f}, t.cleanup...)
}

func (t *MainT) RunCleanup() {
	for _, fn := range t.cleanup {
		fn()
	}
}

func (t MainT) Fatal(a ...any) {
	t.Log(a...)
	t.FailNow()
}

func (t MainT) Skip(a ...any) {
	panic("skip not supported")
}

func (t MainT) Helper() {}

// FailNow exits with a non-zero code
func (t MainT) FailNow() {
	os.Exit(1)
}

// Fail exits with a non-zero code
func (t MainT) Fail() {
	os.Exit(2)
}

// Log args by printing them to stdout
func (t MainT) Log(args ...interface{}) {
	fmt.Println(args...)
}
