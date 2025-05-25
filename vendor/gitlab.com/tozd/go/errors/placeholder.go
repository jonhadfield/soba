package errors

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
)

type placeholderStackTracer interface {
	StackTrace() placeholderStack
}

// UnmarshalJSON unnmarshals JSON errors into placeholder errors
// which can then be formatted in the same way as other errors
// from this package.
//
// Placeholder errors contain same data as original errors (those
// marshaled into JSON), but have multiple limitations:
//
//  1. They do not implement stackTracer interface because addresses
//     of stack frames are not available in JSON nor they are portable.
//  2. Placeholder errors are not of the same type as original errors.
//     Thus errors.Is and errors.As do not work.
//  3. The implementation of fmt.Formatter interface of the original error
//     is not used when formatting placeholder errors.
//
// Placeholder errors also have different trees of wrapping errors than
// original errors because during JSON marshal potentially multiple levels
// of wrapping are combined into one JSON object. Nested objects happen
// only for errors implementing causer or unwrapper interface returning
// multiple errors.
func UnmarshalJSON(data []byte) (error, E) { //nolint:revive,stylecheck
	if bytes.Equal(data, []byte("null")) {
		return nil, nil //nolint:nilnil
	}
	var payload map[string]json.RawMessage
	err := json.Unmarshal(data, &payload)
	if err != nil {
		return nil, WithStack(err)
	}

	var errE E
	var msg string
	var s placeholderStack
	var errs []error
	var cause error

	errorData, ok := payload["error"]
	delete(payload, "error")
	if ok {
		err := json.Unmarshal(errorData, &msg)
		if err != nil {
			return nil, WithMessage(err, "error")
		}
	}

	stackData, ok := payload["stack"]
	delete(payload, "stack")
	if ok {
		err := json.Unmarshal(stackData, &s)
		if err != nil {
			return nil, WithMessage(err, "stack")
		}
		if len(s) == 0 {
			s = nil
		}
	}

	causeData, ok := payload["cause"]
	delete(payload, "cause")
	if ok {
		cause, errE = UnmarshalJSON(causeData)
		if errE != nil {
			return nil, WithMessage(errE, "cause")
		}
	}

	errorsData, ok := payload["errors"]
	delete(payload, "errors")
	if ok {
		var errorsSliceData []json.RawMessage
		err := json.Unmarshal(errorsData, &errorsSliceData)
		if err != nil {
			return nil, WithMessage(err, "errors")
		}
		for i, d := range errorsSliceData {
			e, errE := UnmarshalJSON(d)
			if errE != nil {
				return nil, WithMessagef(errE, "errors: %d", i)
			}
			if e != nil {
				// If e is equal to cause, we want to have e be the same pointer to cause, so that
				// handling of wrapError-like errors can be simplified in formatting and JSON marshal.
				if cause != nil && reflect.DeepEqual(e, cause) { //nolint:govet
					errs = append(errs, cause)
				} else {
					errs = append(errs, e)
				}
			}
		}
		if len(errs) == 0 {
			errs = nil
		}
	}

	details := map[string]interface{}{}
	for key, value := range payload {
		var v interface{}
		err := json.Unmarshal(value, &v)
		if err != nil {
			return nil, WithMessage(err, key)
		}
		details[key] = v
	}

	if cause != nil && len(errs) > 0 {
		return &placeholderJoinedCauseError{
			msg:     msg,
			stack:   s,
			details: details,
			cause:   cause,
			errs:    errs,
		}, nil
	} else if cause != nil {
		return &placeholderCauseError{
			msg:     msg,
			stack:   s,
			details: details,
			cause:   cause,
		}, nil
	} else if len(errs) > 0 {
		return &placeholderJoinedError{
			msg:     msg,
			stack:   s,
			details: details,
			errs:    errs,
		}, nil
	}
	return &placeholderError{
		msg:     msg,
		stack:   s,
		details: details,
	}, nil
}

type placeholderFrame struct {
	Name string `json:"name,omitempty"`
	File string `json:"file,omitempty"`
	Line int    `json:"line,omitempty"`
}

type placeholderStack []placeholderFrame

func (s placeholderStack) Format(st fmt.State, verb rune) {
	for _, f := range s {
		frame{ //nolint:exhaustruct
			Function: f.Name,
			Line:     f.Line,
			File:     f.File,
		}.Format(st, verb)
		_, _ = io.WriteString(st, "\n")
	}
}

type placeholderError struct {
	msg     string
	stack   placeholderStack
	details map[string]interface{}
}

func (e *placeholderError) Error() string {
	if isCalledFromRuntimePanic() {
		return fmt.Sprintf("% -+#.1v", Formatter{Error: e})
	}
	return e.msg
}

func (e *placeholderError) Format(s fmt.State, verb rune) {
	fmt.Fprintf(s, formatString(s, verb), Formatter{Error: e})
}

func (e placeholderError) MarshalJSON() ([]byte, error) {
	return marshalJSONError(&e)
}

func (e *placeholderError) StackTrace() placeholderStack {
	return e.stack
}

func (e *placeholderError) Details() map[string]interface{} {
	return e.details
}

type placeholderCauseError struct {
	msg     string
	stack   placeholderStack
	details map[string]interface{}
	cause   error
}

func (e *placeholderCauseError) Error() string {
	if isCalledFromRuntimePanic() {
		return fmt.Sprintf("% -+#.1v", Formatter{Error: e})
	}
	return e.msg
}

func (e *placeholderCauseError) Format(s fmt.State, verb rune) {
	fmt.Fprintf(s, formatString(s, verb), Formatter{Error: e})
}

func (e placeholderCauseError) MarshalJSON() ([]byte, error) {
	return marshalJSONError(&e)
}

func (e *placeholderCauseError) StackTrace() placeholderStack {
	return e.stack
}

func (e *placeholderCauseError) Details() map[string]interface{} {
	return e.details
}

func (e *placeholderCauseError) Unwrap() error {
	return e.cause
}

func (e *placeholderCauseError) Cause() error {
	return e.cause
}

type placeholderJoinedError struct {
	msg     string
	stack   placeholderStack
	details map[string]interface{}
	errs    []error
}

func (e *placeholderJoinedError) Error() string {
	if isCalledFromRuntimePanic() {
		return fmt.Sprintf("% -+#.1v", Formatter{Error: e})
	}
	return e.msg
}

func (e *placeholderJoinedError) Format(s fmt.State, verb rune) {
	fmt.Fprintf(s, formatString(s, verb), Formatter{Error: e})
}

func (e placeholderJoinedError) MarshalJSON() ([]byte, error) {
	return marshalJSONError(&e)
}

func (e *placeholderJoinedError) StackTrace() placeholderStack {
	return e.stack
}

func (e *placeholderJoinedError) Details() map[string]interface{} {
	return e.details
}

func (e *placeholderJoinedError) Unwrap() []error {
	return e.errs
}

type placeholderJoinedCauseError struct {
	msg     string
	stack   placeholderStack
	details map[string]interface{}
	cause   error
	errs    []error
}

func (e *placeholderJoinedCauseError) Error() string {
	if isCalledFromRuntimePanic() {
		return fmt.Sprintf("% -+#.1v", Formatter{Error: e})
	}
	return e.msg
}

func (e *placeholderJoinedCauseError) Format(s fmt.State, verb rune) {
	fmt.Fprintf(s, formatString(s, verb), Formatter{Error: e})
}

func (e placeholderJoinedCauseError) MarshalJSON() ([]byte, error) {
	return marshalJSONError(&e)
}

func (e *placeholderJoinedCauseError) StackTrace() placeholderStack {
	return e.stack
}

func (e *placeholderJoinedCauseError) Details() map[string]interface{} {
	return e.details
}

func (e *placeholderJoinedCauseError) Unwrap() []error {
	return e.errs
}

func (e *placeholderJoinedCauseError) Cause() error {
	return e.cause
}
