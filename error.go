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
	"log"
	"reflect"
	"runtime"
)

// The maximum number of stackframes on any error.
var MaxStackDepth = 50

// Error is an error with an attached stacktrace. It can be used
// wherever the builtin error interface is expected.
type Error struct {
	Err         error  `json:"error"`
	Stack       string `json:"stack"`
	Description string `json:"description"`
	Title       string `json:"title"`
	stack       []uintptr
	frames      []StackFrame
}

// New makes an Error from the given value. If that value is already an
// error then it will be used directly, if not, it will be passed to
// fmt.Errorf("%v"). The stacktrace will point to the line of code that
// called New.
func New(e interface{}) *Error {
	var err error

	switch e := e.(type) {
	case error:
		err = e
	default:
		err = fmt.Errorf("%v", e)
	}

	s := make([]uintptr, MaxStackDepth)
	length := runtime.Callers(2, s[:])
	f := stackFrames(s[:length])
	return &Error{
		Err:         err,
		stack:       s[:length],
		frames:      f,
		Title:       err.Error(),
		Stack:       errorStack(typeName(err), err, stack(f)),
		Description: err.Error(),
	}
}

// Custom is the same as New but allows you to override the Description and
// Title fields of the Error Struct.
func Custom(e interface{}, desc, title string) *Error {
	var err error
	switch e := e.(type) {
	case error:
		err = e
	default:
		err = fmt.Errorf("%v", e)
	}
	s := make([]uintptr, MaxStackDepth)
	length := runtime.Callers(2, s[:])
	f := stackFrames(s[:length])
	return &Error{
		Err:         err,
		stack:       s[:length],
		Title:       title,
		Description: desc,
		Stack:       errorStack(typeName(err), err, stack(f)),
	}
}

// Wrap makes an Error from the given value. If that value is already an
// error then it will be used directly, if not, it will be passed to
// fmt.Errorf("%v"). The skip parameter indicates how far up the stack
// to start the stacktrace. 0 is from the current call, 1 from its caller, etc.
func Wrap(e interface{}, skip int) *Error {
	var err error

	switch e := e.(type) {
	case *Error:
		return e
	case error:
		err = e
	default:
		err = fmt.Errorf("%v", e)
	}

	s := make([]uintptr, MaxStackDepth)
	length := runtime.Callers(2+skip, s[:])
	f := stackFrames(s[:length])
	return &Error{
		Err:    err,
		stack:  s[:length],
		frames: f,
		//	StackTrace: string(stack(f)),
		Stack: errorStack(typeName(err), err, stack(f)),
	}
}

// Is detects whether the error is equal to a given error. Errors
// are considered equal by this function if they are the same object,
// or if they both contain the same error inside an errors.Error.
func Is(e error, original error) bool {

	if e == original {
		return true
	}

	if e, ok := e.(*Error); ok {
		return Is(e.Err, original)
	}

	if original, ok := original.(*Error); ok {
		return Is(e, original.Err)
	}

	return false
}

// Errorf creates a new error with the given message. You can use it
// as a drop-in replacement for fmt.Errorf() to provide descriptive
// errors in return values.
func Errorf(format string, a ...interface{}) *Error {
	return Wrap(fmt.Errorf(format, a...), 1)
}

// Error returns the underlying error's message.
func (err *Error) Error() string {
	return err.Err.Error()
}

// StackTrace returns the callstack formatted the same way that go does
// in runtime/debug.Stack()
func (err *Error) StackTrace() []byte {
	if err.frames == nil {
		err.frames = err.StackFrames()
	}
	return stack(err.frames)
}

func stack(frames []StackFrame) []byte {
	buf := bytes.Buffer{}

	for _, frame := range frames {
		_, err := buf.WriteString(frame.String())
		if err != nil {
			log.Panic(err)
		}
	}

	return buf.Bytes()
}

// ErrorStack returns a string that contains both the
// error message and the callstack.
func (err *Error) ErrorStack() string {
	return errorStack(err.TypeName(), err, err.StackTrace())
}

func errorStack(t string, err error, s []byte) string {
	return t + " " + err.Error() + "\n" + string(s)
}

func stackFrames(stack []uintptr) []StackFrame {
	frames := make([]StackFrame, len(stack))
	for i, pc := range stack {
		frames[i] = NewStackFrame(pc)
	}
	return frames
}

// StackFrames returns an array of frames containing information about the
// stack.
func (err *Error) StackFrames() []StackFrame {
	if err.frames == nil {
		err.frames = stackFrames(err.stack)
	}

	return err.frames
}

// TypeName returns the type this error. e.g. *errors.stringError.
func (err *Error) TypeName() string {
	return typeName(err.Err)
}

func typeName(err error) string {
	if _, ok := err.(uncaughtPanic); ok {
		return "panic"
	}
	return reflect.TypeOf(err).String()
}
