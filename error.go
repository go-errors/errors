// Package errors provides errors that have stack-traces.
//
// This is particularly useful when you want to understand the
// state of execution when an error was returned unexpectedly.
//
// It provides the type *Error which implements the standard
// golang error interface, so you can use this library interchangably
// with code that is expecting a normal error return.
//
// For example:
//
//  package crashy
//
//  import "github.com/go-errors/errors"
//
//  var Crashed = errors.Errorf("oh dear")
//
//  func Crash() error {
//      return errors.New(Crashed)
//  }
//
// This can be called as follows:
//
//  package main
//
//  import (
//      "crashy"
//      "fmt"
//      "github.com/go-errors/errors"
//  )
//
//  func main() {
//      err := crashy.Crash()
//      if err != nil {
//          if errors.Is(err, crashy.Crashed) {
//              fmt.Println(err.(*errors.Error).ErrorStack())
//          } else {
//              panic(err)
//          }
//      }
//  }
//
// This package was original written to allow reporting to Bugsnag,
// but after I found similar packages by Facebook and Dropbox, it
// was moved to one canonical location so everyone can benefit.
package errors

import (
	"bytes"
	"fmt"
	"reflect"
	"runtime"
)

// The maximum number of stackframes on any error.
var MaxStackDepth = 50

// Error is an error with an attached stacktrace. It can be used
// wherever the builtin error interface is expected.
type Error interface {
	error
	Underlying() error
	Stack() []byte
	ErrorStack() string
	StackFrames() []StackFrame
	TypeName() string
}

// wrappedError is an implementation of the Error interface.
type wrappedError struct {
	err    error
	stack  []uintptr
	frames []StackFrame
	prefix string
}

// New makes an Error from the given value. If that value is already an
// error then it will be used directly, if not, it will be passed to
// fmt.Errorf("%v"). The stacktrace will point to the line of code that
// called New.
func New(e interface{}) Error {
	var err error

	switch e := e.(type) {
	case error:
		err = e
	default:
		err = fmt.Errorf("%v", e)
	}

	stack := make([]uintptr, MaxStackDepth)
	length := runtime.Callers(2, stack[:])
	return &wrappedError{
		err:   err,
		stack: stack[:length],
	}
}

// Wrap makes an Error from the given value. If that value is already an
// error then it will be used directly, if not, it will be passed to
// fmt.Errorf("%v"). The skip parameter indicates how far up the stack
// to start the stacktrace. 0 is from the current call, 1 from its caller, etc.
func Wrap(e interface{}, skip int) Error {
	var err error

	switch e := e.(type) {
	case *wrappedError:
		return e
	case error:
		err = e
	default:
		err = fmt.Errorf("%v", e)
	}

	stack := make([]uintptr, MaxStackDepth)
	length := runtime.Callers(2+skip, stack[:])
	return &wrappedError{
		err:   err,
		stack: stack[:length],
	}
}

// WrapPrefix makes an Error from the given value. If that value is already an
// error then it will be used directly, if not, it will be passed to
// fmt.Errorf("%v"). The prefix parameter is used to add a prefix to the
// error message when calling Error(). The skip parameter indicates how far
// up the stack to start the stacktrace. 0 is from the current call,
// 1 from its caller, etc.
func WrapPrefix(e interface{}, prefix string, skip int) Error {

	err := Wrap(e, skip).(*wrappedError)

	if err.prefix != "" {
		err.prefix = fmt.Sprintf("%s: %s", prefix, err.prefix)
	} else {
		err.prefix = prefix
	}

	return err

}

// Is detects whether the error is equal to a given error. Errors
// are considered equal by this function if they are the same object,
// or if they both contain the same error inside an errors.Error.
func Is(e error, original error) bool {

	if e == original {
		return true
	}

	if e, ok := e.(*wrappedError); ok {
		return Is(e.err, original)
	}

	if original, ok := original.(*wrappedError); ok {
		return Is(e, original.err)
	}

	return false
}

// Errorf creates a new error with the given message. You can use it
// as a drop-in replacement for fmt.Errorf() to provide descriptive
// errors in return values.
func Errorf(format string, a ...interface{}) Error {
	return Wrap(fmt.Errorf(format, a...), 1)
}

// Underlying returns the underlying error.
func (err *wrappedError) Underlying() error {
	return err.err
}

// Error returns the underlying error's message.
func (err wrappedError) Error() string {

	msg := err.err.Error()
	if err.prefix != "" {
		msg = fmt.Sprintf("%s: %s", err.prefix, msg)
	}

	return msg
}

// Stack returns the callstack formatted the same way that go does
// in runtime/debug.Stack()
func (err wrappedError) Stack() []byte {
	buf := bytes.Buffer{}

	for _, frame := range err.StackFrames() {
		buf.WriteString(frame.String())
	}

	return buf.Bytes()
}

// Callers satisfies the bugsnag ErrorWithCallerS() interface
// so that the stack can be read out.
func (err *Error) Callers() []uintptr {
	return err.stack
}

// ErrorStack returns a string that contains both the
// error message and the callstack.
func (err wrappedError) ErrorStack() string {
	return err.TypeName() + " " + err.Error() + "\n" + string(err.Stack())
}

// StackFrames returns an array of frames containing information about the
// stack.
func (err wrappedError) StackFrames() []StackFrame {
	if err.frames == nil {
		err.frames = make([]StackFrame, len(err.stack))

		for i, pc := range err.stack {
			err.frames[i] = NewStackFrame(pc)
		}
	}

	return err.frames
}

// TypeName returns the type this error. e.g. *errors.stringError.
func (err wrappedError) TypeName() string {
	if _, ok := err.err.(uncaughtPanic); ok {
		return "panic"
	}
	return reflect.TypeOf(err.err).String()
}
