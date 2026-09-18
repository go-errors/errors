//go:build !go1.20
// +build !go1.20

package errors

import (
	"reflect"
)

// Before go1.20 the stdlib errors.Is and errors.As only understand
// Unwrap() error, so they cannot traverse the errors returned by Join, which
// this package backports in join_unwrap_backward.go. The implementations below
// are used instead, so that joined errors resolve the same way on every go
// version. This file must not import the stdlib errors package, as it has no
// Is and As before go1.13.
//
// Disclaimer: functions baseIs, is, baseAs and as, and the errorType variable,
// are copied from the stdlib errors package v1.21.0, with internal/reflectlite
// replaced by reflect and any replaced by interface{}.

// baseIs reports whether any error in err's tree matches target.
//
// The tree consists of err itself, followed by the errors obtained by repeatedly
// calling its Unwrap() error or Unwrap() []error method. When err wraps multiple
// errors, baseIs examines err followed by a depth-first traversal of its
// children.
//
// An error is considered to match a target if it is equal to that target or if
// it implements a method Is(error) bool such that Is(target) returns true.
func baseIs(err, target error) bool {
	if err == nil || target == nil {
		return err == target
	}

	isComparable := reflect.TypeOf(target).Comparable()
	return is(err, target, isComparable)
}

func is(err, target error, targetComparable bool) bool {
	for {
		if targetComparable && err == target {
			return true
		}
		if x, ok := err.(interface{ Is(error) bool }); ok && x.Is(target) {
			return true
		}
		switch x := err.(type) {
		case interface{ Unwrap() error }:
			err = x.Unwrap()
			if err == nil {
				return false
			}
		case interface{ Unwrap() []error }:
			for _, err := range x.Unwrap() {
				if is(err, target, targetComparable) {
					return true
				}
			}
			return false
		default:
			return false
		}
	}
}

// baseAs finds the first error in err's tree that matches target, and if one is
// found, sets target to that error value and returns true. Otherwise, it returns
// false.
//
// The tree consists of err itself, followed by the errors obtained by repeatedly
// calling its Unwrap() error or Unwrap() []error method. When err wraps multiple
// errors, baseAs examines err followed by a depth-first traversal of its
// children.
//
// An error matches target if the error's concrete value is assignable to the
// value pointed to by target, or if the error has a method
// As(interface{}) bool such that As(target) returns true. In the latter case,
// the As method is responsible for setting target.
//
// baseAs panics if target is not a non-nil pointer to either a type that
// implements error, or to any interface type.
func baseAs(err error, target interface{}) bool {
	if err == nil {
		return false
	}
	if target == nil {
		panic("errors: target cannot be nil")
	}
	val := reflect.ValueOf(target)
	typ := val.Type()
	if typ.Kind() != reflect.Ptr || val.IsNil() {
		panic("errors: target must be a non-nil pointer")
	}
	targetType := typ.Elem()
	if targetType.Kind() != reflect.Interface && !targetType.Implements(errorType) {
		panic("errors: *target must be interface or implement error")
	}
	return as(err, target, val, targetType)
}

func as(err error, target interface{}, targetVal reflect.Value, targetType reflect.Type) bool {
	for {
		if reflect.TypeOf(err).AssignableTo(targetType) {
			targetVal.Elem().Set(reflect.ValueOf(err))
			return true
		}
		if x, ok := err.(interface{ As(interface{}) bool }); ok && x.As(target) {
			return true
		}
		switch x := err.(type) {
		case interface{ Unwrap() error }:
			err = x.Unwrap()
			if err == nil {
				return false
			}
		case interface{ Unwrap() []error }:
			for _, err := range x.Unwrap() {
				if err == nil {
					continue
				}
				if as(err, target, targetVal, targetType) {
					return true
				}
			}
			return false
		default:
			return false
		}
	}
}

var errorType = reflect.TypeOf((*error)(nil)).Elem()
