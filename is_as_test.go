package errors

import (
	"fmt"
	"io"
	"testing"
)

var (
	errFirst  = fmt.Errorf("first")
	errSecond = fmt.Errorf("second")
	errThird  = fmt.Errorf("third")
)

// Go 1.20 did not only add errors.Join, it also taught errors.Is and errors.As
// to traverse errors implementing Unwrap() []error. Since this package
// backports Join to older versions, it has to backport that traversal too,
// otherwise joined errors resolve differently on go1.13..go1.19 than they do on
// go1.20 and later. The tests below pin the go1.20+ behaviour on every version.

func TestIsJoined(t *testing.T) {
	joined := Join(errFirst, errSecond)

	if !Is(joined, errFirst) {
		t.Errorf("joined error is not errFirst")
	}

	if !Is(joined, errSecond) {
		t.Errorf("joined error is not errSecond")
	}

	if Is(joined, errThird) {
		t.Errorf("joined error is errThird")
	}
}

func TestIsNestedJoined(t *testing.T) {
	joined := Join(errFirst, Join(errSecond, Join(errThird, io.EOF)))

	if !Is(joined, errThird) {
		t.Errorf("nested joined error is not errThird")
	}

	if !Is(joined, io.EOF) {
		t.Errorf("nested joined error is not io.EOF")
	}

	if Is(joined, io.ErrUnexpectedEOF) {
		t.Errorf("nested joined error is io.ErrUnexpectedEOF")
	}
}

// TestIsJoinedError covers the *Error unwrapping this package adds on top of the
// stdlib behaviour, on both the error and the target side of a join.
func TestIsJoinedError(t *testing.T) {
	joined := Join(errFirst, New(io.EOF))

	if !Is(joined, io.EOF) {
		t.Errorf("joined New(io.EOF) is not io.EOF")
	}

	if !Is(joined, New(io.EOF)) {
		t.Errorf("joined New(io.EOF) is not New(io.EOF)")
	}

	if Is(joined, New(io.ErrUnexpectedEOF)) {
		t.Errorf("joined New(io.EOF) is New(io.ErrUnexpectedEOF)")
	}

	if !Is(New(Join(errFirst, errSecond)), errSecond) {
		t.Errorf("New(joined error) is not errSecond")
	}
}

func TestIsJoinedCustomIs(t *testing.T) {
	joined := Join(errFirst, joinCustomIs{key: "match"})

	if !Is(joined, joinCustomIs{key: "match"}) {
		t.Errorf("joined error is not a match joinCustomIs")
	}

	if Is(joined, joinCustomIs{key: "other"}) {
		t.Errorf("joined error is an other joinCustomIs")
	}
}

// TestIsUncomparableTarget makes sure an uncomparable target is not compared
// with ==, which would panic at runtime.
func TestIsUncomparableTarget(t *testing.T) {
	joined := Join(errFirst, uncomparableError{msgs: []string{"uncomparable"}})

	if Is(joined, uncomparableError{msgs: []string{"uncomparable"}}) {
		t.Errorf("joined error is an uncomparableError")
	}
}

func TestIsJoinedNil(t *testing.T) {
	if err := Join(nil, nil); err != nil {
		t.Errorf("join of nil errors is not nil: %v", err)
	}

	if !Is(Join(nil, errFirst), errFirst) {
		t.Errorf("join with a nil error dropped errFirst")
	}

	if Is(Join(errFirst, errSecond), nil) {
		t.Errorf("joined error is nil")
	}

	if Is(nil, errFirst) {
		t.Errorf("nil is errFirst")
	}
}

func TestAsJoined(t *testing.T) {
	joined := Join(errFirst, Wrap(errorString("TestForFun"), 0))

	var errStrOut errorString
	if !As(joined, &errStrOut) {
		t.Errorf("joined errorString is not returned")
	} else if errStrOut != "TestForFun" {
		t.Errorf("joined errorString value is not returned")
	}

	var errOut *Error
	if !As(joined, &errOut) {
		t.Errorf("joined *Error is not returned")
	}

	var stackOut stackTracer
	if !As(joined, &stackOut) {
		t.Errorf("joined stackTracer is not returned")
	}

	var missing joinCustomAsTarget
	if As(Join(errFirst, errSecond), &missing) {
		t.Errorf("joined error matched an absent target")
	}

	if As(nil, &errStrOut) {
		t.Errorf("nil matched a target")
	}
}

func TestAsJoinedCustomAs(t *testing.T) {
	joined := Join(errFirst, joinCustomAs{payload: "payload"})

	var out joinCustomAsTarget
	if !As(joined, &out) {
		t.Errorf("joined joinCustomAs is not returned")
	} else if out.payload != "payload" {
		t.Errorf("joined joinCustomAs value is not returned")
	}
}

// TestAsJoinedNilMember covers an Unwrap() []error returning a slice with a nil
// element, which must be skipped instead of panicking.
func TestAsJoinedNilMember(t *testing.T) {
	err := nilMemberError{errs: []error{nil, New(io.EOF)}}

	var errOut *Error
	if !As(err, &errOut) {
		t.Errorf("*Error next to a nil member is not returned")
	}

	if !Is(err, io.EOF) {
		t.Errorf("io.EOF next to a nil member is not found")
	}
}

// TestUnwrapJoined pins that Unwrap does not unwrap the errors returned by Join,
// on any go version.
func TestUnwrapJoined(t *testing.T) {
	if err := Unwrap(Join(errFirst, errSecond)); err != nil {
		t.Errorf("Unwrap unwrapped a joined error: %v", err)
	}

	if err := Unwrap(New(io.EOF)); err != io.EOF {
		t.Errorf("Unwrap did not unwrap an *Error: %v", err)
	}
}

type stackTracer interface {
	StackFrames() []StackFrame
}

type joinCustomIs struct {
	key string
}

func (e joinCustomIs) Error() string { return "joinCustomIs: " + e.key }

func (e joinCustomIs) Is(target error) bool {
	matched, ok := target.(joinCustomIs)
	return ok && matched.key == e.key
}

type joinCustomAs struct {
	payload string
}

func (e joinCustomAs) Error() string { return "joinCustomAs: " + e.payload }

func (e joinCustomAs) As(target interface{}) bool {
	matched, ok := target.(*joinCustomAsTarget)
	if !ok {
		return false
	}
	matched.payload = e.payload
	return true
}

type joinCustomAsTarget struct {
	payload string
}

func (t joinCustomAsTarget) Error() string { return "joinCustomAsTarget: " + t.payload }

type uncomparableError struct {
	msgs []string
}

func (e uncomparableError) Error() string { return "uncomparableError" }

type nilMemberError struct {
	errs []error
}

func (e nilMemberError) Error() string { return "nilMemberError" }

func (e nilMemberError) Unwrap() []error { return e.errs }
