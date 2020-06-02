// +build !go1.13

package errors

// Is detects whether the error is equal to a given error. Errors	// Is detects whether the error is equal to a given error. Errors
// are considered equal by this function if they are the same object,	// are considered equal by this function if they are matched by errors.Is
// or if they both contain the same error inside an errors.Error.	// or if their contained errors are matched through errors.Is
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