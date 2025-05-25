package errors

import (
	stderrors "errors"
	"fmt"
)

// Is reports whether any error in err's tree matches target.
//
// The tree consists of err itself, followed by the errors obtained by repeatedly
// calling Unwrap. When err wraps multiple errors, Is examines err followed by a
// depth-first traversal of its children.
//
// An error is considered to match a target if it is equal to that target or if
// it implements a method Is(error) bool such that Is(target) returns true.
//
// An error type might provide an Is method so it can be treated as equivalent
// to an existing error. For example, if MyError defines
//
//	func (m MyError) Is(target error) bool { return target == fs.ErrExist }
//
// then Is(MyError{}, fs.ErrExist) returns true. See [syscall.Errno.Is] for
// an example in the standard library. An Is method should only shallowly
// compare err and the target and not call Unwrap on either.
//
// This function is a proxy for standard errors.Is.
func Is(err, target error) bool {
	return stderrorsIs(err, target)
}

// As finds the first error in err's tree that matches target, and if one is found, sets
// target to that error value and returns true. Otherwise, it returns false.
//
// The tree consists of err itself, followed by the errors obtained by repeatedly
// calling Unwrap. When err wraps multiple errors, As examines err followed by a
// depth-first traversal of its children.
//
// An error matches target if the error's concrete value is assignable to the value
// pointed to by target, or if the error has a method As(interface{}) bool such that
// As(target) returns true. In the latter case, the As method is responsible for
// setting target.
//
// An error type might provide an As method so it can be treated as if it were a
// different error type.
//
// As panics if target is not a non-nil pointer to either a type that implements
// error, or to any interface type.
//
// This function is a proxy for standard errors.As.
func As(err error, target interface{}) bool {
	return stderrorsAs(err, target)
}

// Unwrap returns the result of calling the Unwrap method on err, if err's
// type contains an Unwrap method returning error.
// Otherwise, Unwrap returns nil.
//
// Unwrap only calls a method of the form "Unwrap() error".
// In particular Unwrap does not unwrap errors returned by [Join].
//
// This function is a proxy for standard errors.Unwrap and is not
// an inverse of errors.Wrap. For that use errors.Cause.
func Unwrap(err error) error {
	return stderrors.Unwrap(err)
}

// Base returns an error with the supplied message.
// Each call to Base returns a distinct error value even if the message is identical.
// It does not record a stack trace.
//
// Use Base for a constant base error you convert to an actual error you return with
// WithStack or WithDetails. This base error you can then use in Is and As calls.
//
// This function is a proxy for standard errors.New.
func Base(message string) error {
	return stderrors.New(message)
}

// Basef returns an error with the supplied message
// formatted according to a format specifier.
// Each call to Basef returns a distinct error value even if the message is identical.
// It does not record a stack trace. It supports %w format verb to wrap an existing error.
// %w can be provided multiple times.
//
// Use Basef for a constant base error you convert to an actual error you return with
// WithStack or WithDetails. This base error you can then use in Is and As calls. Use %w format verb
// when you want to create a tree of base errors.
//
// This function is a proxy for standard fmt.Errorf.
func Basef(format string, args ...interface{}) error {
	return fmt.Errorf(format, args...)
}

// BaseWrap returns an error with the supplied message, wrapping an existing error
// err.
// Each call to BaseWrap returns a distinct error value even if the message is identical.
// It does not record a stack trace.
//
// Use BaseWrap when you want to create a tree of base errors and you want to fully
// control the error message.
func BaseWrap(err error, message string) error {
	return &base{
		err,
		message,
	}
}

// BaseWrapf returns an error with the supplied message
// formatted according to a format specifier.
// Each call to BaseWrapf returns a distinct error value even if the message is identical.
// It does not record a stack trace. It does not support %w format verb
// (use %s instead if you need to incorporate error's error message, but then you can
// also just use Basef).
//
// Use BaseWrapf when you want to create a tree of base errors and you want to fully
// control the error message.
func BaseWrapf(err error, format string, args ...interface{}) error {
	return &base{
		err,
		fmt.Sprintf(format, args...),
	}
}

type base struct {
	err error
	msg string
}

func (b *base) Unwrap() error {
	return b.err
}

func (b *base) Error() string {
	return b.msg
}
