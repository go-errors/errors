// +build go1.13

package errors

import (
	"io"
	"testing"
)

// This test should work only for go 1.13 and latter
func TestIs113(t *testing.T) {
	custErr := errorWithCustomIs{
		Key: "TestForFun",
		Err: io.EOF,
	}

	shouldMatch := errorWithCustomIs{
		Key: "TestForFun",
	}

	shouldNotMatch := errorWithCustomIs{Key: "notOk"}

	if !Is(custErr, shouldMatch) {
		t.Errorf("custErr is not a TestForFun customError")
	}

	if Is(custErr, shouldNotMatch) {
		t.Errorf("custErr is a notOk customError")
	}

	if !Is(custErr, New(shouldMatch)) {
		t.Errorf("custErr is not a New(TestForFun customError)")
	}

	if Is(custErr, New(shouldNotMatch)) {
		t.Errorf("custErr is a New(notOk customError)")
	}

	if !Is(New(custErr), shouldMatch) {
		t.Errorf("New(custErr) is not a TestForFun customError")
	}

	if Is(New(custErr), shouldNotMatch) {
		t.Errorf("New(custErr) is a notOk customError")
	}

	if !Is(New(custErr), New(shouldMatch)) {
		t.Errorf("New(custErr) is not a New(TestForFun customError)")
	}

	if Is(New(custErr), New(shouldNotMatch)) {
		t.Errorf("New(custErr) is a New(notOk customError)")
	}
}

type errorWithCustomIs struct {
	Key string
	Err error
}

func (ewci errorWithCustomIs) Error() string {
	return "[" + ewci.Key + "]: " + ewci.Err.Error()
}

func (ewci errorWithCustomIs) Is(target error) bool {
	matched, ok := target.(errorWithCustomIs)
	return ok && matched.Key == ewci.Key
}
