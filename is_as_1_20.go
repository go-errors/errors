//go:build go1.20
// +build go1.20

package errors

import baseErrors "errors"

// baseIs reports whether any error in err's tree matches target. Since go1.20
// the stdlib traverses errors implementing Unwrap() []error, such as the ones
// returned by Join, so it can be used as is.
func baseIs(err, target error) bool {
	return baseErrors.Is(err, target)
}

// baseAs finds the first error in err's tree that matches target. Since go1.20
// the stdlib traverses errors implementing Unwrap() []error, such as the ones
// returned by Join, so it can be used as is.
func baseAs(err error, target interface{}) bool {
	return baseErrors.As(err, target)
}
