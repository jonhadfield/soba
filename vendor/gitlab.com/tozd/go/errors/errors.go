// Package errors provides errors with a recorded stack trace and optional
// structured details.
//
// The traditional error handling idiom in Go is roughly akin to
//
//	if err != nil {
//	        return err
//	}
//
// which when applied recursively up the call stack results in error reports
// without a stack trace or context.
// The errors package provides error handling primitives to annotate
// errors along the failure path in a way that does not destroy the
// original error.
//
// # Adding a stack trace to an error
//
// When interacting with code which returns errors without a stack trace,
// you can upgrade that error to one with a stack trace using errors.WithStack.
// For example:
//
//	func readAll(r io.Reader) ([]byte, errors.E) {
//	        data, err := ioutil.ReadAll(r)
//	        if err != nil {
//	                return nil, errors.WithStack(err)
//	        }
//	        return data, nil
//	}
//
// errors.WithStack records the stack trace at the point where it was called, so
// use it as close to where the error originated as you can get so that the
// recorded stack trace is more precise.
//
// The example above uses errors.E for the returned error type instead of the
// standard error type. This is not required, but it tells Go that you expect
// that the function returns only errors with a stack trace and Go type system
// then helps you find any cases where this is not so.
//
// errors.WithStack does not record the stack trace if it is already present in
// the error so it is safe to call it if you are unsure if the error contains
// a stack trace.
//
// Errors with a stack trace implement the following interface, returning program
// counters of function invocations:
//
//	type stackTracer interface {
//	        StackTrace() []uintptr
//	}
//
// You can use standard runtime.CallersFrames to obtain stack trace frame
// information (e.g., function name, source code file and line).
// You can also use errors.StackFormatter to format the stack trace.
//
// Although the stackTracer interface is not exported by this package, it is
// considered a part of its stable public interface.
//
// # Adding context to an error
//
// Sometimes an error occurs in a low-level function and the error messages
// returned from it are too low-level, too.
// You can use errors.Wrap to construct a new higher-level error while
// recording the original error as a cause.
//
//	image, err := readAll(imageFile)
//	if err != nil {
//	        return nil, errors.Wrap(err, "reading image failed")
//	}
//
// In the example above we returned a new error with a new message,
// hiding the low-level details. The returned error implements the
// following interface:
//
//	type causer interface {
//	        Cause() error
//	}
//
// which enables access to the underlying low-level error. You can also
// use errors.Cause to obtain the cause.
//
// Although the causer interface is not exported by this package, it is
// considered a part of its stable public interface.
//
// Sometimes you do not want to hide the error message but just add to it.
// You can use errors.WithMessage, which adds a prefix to the existing message,
// or errors.Errorf, which gives you more control over the new message.
//
//	errors.WithMessage(err, "reading image failed")
//	errors.Errorf("reading image failed (%w)", err)
//
// Example new messages could then be, respectively:
//
//	"reading image failed: connection error"
//	"reading image failed (connection error)"
//
// # Adding details to an error
//
// Errors returned by this package implement the detailer interface:
//
//	type detailer interface {
//	        Details() map[string]interface{}
//	}
//
// which enables access to a map with optional additional details about
// the error. Returned map can be modified in-place. You can also use
// errors.Details and errors.AllDetails to access details:
//
//	errors.Details(err)["url"] = "http://example.com"
//
// You can also use errors.WithDetails as an alternative to errors.WithStack
// if you also want to add details while recording the stack trace:
//
//	func readAll(r io.Reader, filename string) ([]byte, errors.E) {
//	        data, err := ioutil.ReadAll(r)
//	        if err != nil {
//	                return nil, errors.WithDetails(err, "filename", filename)
//	        }
//	        return data, nil
//	}
//
// # Working with the tree of errors
//
// Errors which implement the following standard unwrapper interfaces:
//
//	type unwrapper interface {
//	        Unwrap() error
//	}
//
//	type unwrapper interface {
//	        Unwrap() error[]
//	}
//
// form a tree of errors where a wrapping error points its parent,
// wrapped, error(s). Errors returned from this package implement this
// interface to return the original error or errors, when they exist.
// This enables us to have constant base errors which we annotate
// with a stack trace before we return them:
//
//	var ErrAuthentication = errors.Base("authentication error")
//	var ErrMissingPassphrase = errors.BaseWrap(ErrAuthentication, "missing passphrase")
//	var ErrInvalidPassphrase = errors.BaseWrap(ErrAuthentication, "invalid passphrase")
//
//	func authenticate(passphrase string) errors.E {
//	        if passphrase == "" {
//	                return errors.WithStack(ErrMissingPassphrase)
//	        } else if passphrase != "open sesame" {
//	                return errors.WithStack(ErrInvalidPassphrase)
//	        }
//	        return nil
//	}
//
// Or with details:
//
//	func authenticate(username, passphrase string) errors.E {
//	        if passphrase == "" {
//	                return errors.WithDetails(ErrMissingPassphrase, "username", username)
//	        } else if passphrase != "open sesame" {
//	                return errors.WithDetails(ErrInvalidPassphrase, "username", username)
//	        }
//	        return nil
//	}
//
// We can use errors.Is to determine which error has been returned:
//
//	if errors.Is(err, ErrMissingPassphrase) {
//	        fmt.Println("Please provide a passphrase to unlock the doors.")
//	}
//
// Works across the tree, too:
//
//	if errors.Is(err, ErrAuthentication) {
//	        fmt.Println("Failed to unlock the doors.")
//	}
//
// To access details, use:
//
//	errors.AllDetails(err)["username"]
//
// You can join multiple errors into one error by calling errors.Join.
// Join also records the stack trace at the point it was called.
//
// # Formatting and JSON marshaling errors
//
// All errors with a stack trace returned from this package implement fmt.Formatter
// interface and can be formatted by the fmt package. They also support marshaling
// to JSON. Same formatting and JSON marshaling for errors coming outside of
// this package can be done by wrapping them into errors.Formatter.
package errors

import (
	"fmt"
	"strings"
	"sync"
	"unsafe"

	pkgerrors "github.com/pkg/errors"
)

type stackTracer interface {
	StackTrace() []uintptr
}

type pkgStackTracer interface {
	StackTrace() pkgerrors.StackTrace
}

type goErrorsStackTracer interface {
	Callers() []uintptr
}

type erisStackTracer interface {
	StackFrames() []uintptr
}

type causer interface {
	Cause() error
}

type unwrapper interface {
	Unwrap() error
}

type unwrapperJoined interface {
	Unwrap() []error
}

type detailer interface {
	Details() map[string]interface{}
}

func getExistingStackTrace(err error) []uintptr {
	for err != nil {
		switch e := err.(type) { //nolint:errorlint
		case stackTracer:
			return e.StackTrace()
		case pkgStackTracer:
			st := e.StackTrace()
			return *(*[]uintptr)(unsafe.Pointer(&st))
		case goErrorsStackTracer:
			return e.Callers()
		case erisStackTracer:
			return e.StackFrames()
		}
		c, ok := err.(causer)
		if ok && c.Cause() != nil {
			return nil
		}
		e, ok := err.(unwrapperJoined)
		if ok && len(e.Unwrap()) > 0 {
			return nil
		}
		err = Unwrap(err)
	}
	return nil
}

// prefixMessage eagerly builds a new message with the provided prefixes.
// This is a trade-off which consumes more memory but allows one to cheaply
// call Error multiple times.
func prefixMessage(msg string, prefixes ...string) string {
	message := strings.Builder{}
	for i, prefix := range prefixes {
		if len(prefix) > 0 {
			message.WriteString(prefix)
			if prefix[len(prefix)-1] != '\n' && (i < len(prefixes)-1 || len(msg) > 0) {
				message.WriteString(": ")
			}
		}
	}
	if len(msg) > 0 {
		message.WriteString(msg)
	}
	return message.String()
}

// This is a trade-off which consumes more memory but allows one to cheaply
// call Error multiple times.
func joinMessages(errs []error) string {
	// Same implementation as standard library's joinError's Error.
	var b []byte
	for i, err := range errs {
		if i > 0 {
			b = append(b, '\n')
		}
		b = append(b, err.Error()...)
	}
	return string(b)
}

// E interface can be used in as a return type instead of the standard error
// interface to annotate which functions return an error with a stack trace
// and details.
// This is useful so that you know when you should use WithStack or WithDetails
// (for functions which do not return E) and when not (for functions which do
// return E).
//
// If you call WithStack on an error with a stack trace nothing bad happens
// (same error is simply returned), it just pollutes the code. So this
// interface is defined to help. (Calling WithDetails on an error with details
// adds an additional and independent layer of details on
// top of any existing details.)
type E interface {
	error
	stackTracer
	detailer
}

// New returns an error with the supplied message.
// New also records the stack trace at the point it was called.
func New(message string) E {
	return &fundamentalError{
		msg:       message,
		stack:     callers(0),
		details:   nil,
		detailsMu: new(sync.Mutex),
	}
}

// Errorf return an error with the supplied message
// formatted according to a format specifier.
// It supports %w format verb to wrap an existing error.
// Errorf also records the stack trace at the point it was called,
// unless wrapped error already have a stack trace.
// If %w is provided multiple times, then a stack trace is always recorded.
func Errorf(format string, args ...interface{}) E {
	err := fmt.Errorf(format, args...) //nolint:err113
	var errs []error
	// Errorf itself maybe wrapped an error or errors so we can use a type switch here
	// and do not need to (and should not) use As to determine if that happened.
	switch u := err.(type) { //nolint:errorlint
	case unwrapperJoined:
		errs = u.Unwrap()
	case unwrapper:
		errs = []error{u.Unwrap()}
	}
	if len(errs) > 1 {
		return &msgJoinedError{
			errs:      errs,
			msg:       err.Error(),
			stack:     callers(0),
			details:   nil,
			detailsMu: new(sync.Mutex),
		}
	} else if len(errs) == 1 {
		unwrap := errs[0]
		st := getExistingStackTrace(unwrap)
		if len(st) == 0 {
			st = callers(0)
		}

		return &msgError{
			err:       unwrap,
			msg:       err.Error(),
			stack:     st,
			details:   nil,
			detailsMu: new(sync.Mutex),
		}
	}

	return &fundamentalError{
		msg:       err.Error(),
		stack:     callers(0),
		details:   nil,
		detailsMu: new(sync.Mutex),
	}
}

// fundamentalError is an error that has a message and a stack,
// but does not wrap another error.
type fundamentalError struct {
	msg       string
	stack     []uintptr
	details   map[string]interface{}
	detailsMu *sync.Mutex
}

func (e *fundamentalError) Error() string {
	if isCalledFromRuntimePanic() {
		return fmt.Sprintf("% -+#.1v", Formatter{Error: e})
	}
	return e.msg
}

func (e *fundamentalError) Format(s fmt.State, verb rune) {
	fmt.Fprintf(s, formatString(s, verb), Formatter{Error: e})
}

func (e fundamentalError) MarshalJSON() ([]byte, error) {
	return marshalJSONError(&e)
}

func (e *fundamentalError) StackTrace() []uintptr {
	return e.stack
}

func (e *fundamentalError) Details() map[string]interface{} {
	e.detailsMu.Lock()
	defer e.detailsMu.Unlock()

	if e.details == nil {
		e.details = make(map[string]interface{})
	}
	return e.details
}

// msgError wraps another error and has its own stack and msg.
type msgError struct {
	err       error
	msg       string
	stack     []uintptr
	details   map[string]interface{}
	detailsMu *sync.Mutex
}

func (e *msgError) Error() string {
	if isCalledFromRuntimePanic() {
		return fmt.Sprintf("% -+#.1v", Formatter{Error: e})
	}
	return e.msg
}

func (e *msgError) Format(s fmt.State, verb rune) {
	fmt.Fprintf(s, formatString(s, verb), Formatter{Error: e})
}

func (e msgError) MarshalJSON() ([]byte, error) {
	return marshalJSONError(&e)
}

func (e *msgError) Unwrap() error {
	return e.err
}

func (e *msgError) StackTrace() []uintptr {
	return e.stack
}

func (e *msgError) Details() map[string]interface{} {
	e.detailsMu.Lock()
	defer e.detailsMu.Unlock()

	if e.details == nil {
		e.details = make(map[string]interface{})
	}
	return e.details
}

// msgJoinedError wraps multiple errors
// and has its own stack and msg.
type msgJoinedError struct {
	errs      []error
	msg       string
	stack     []uintptr
	details   map[string]interface{}
	detailsMu *sync.Mutex
}

func (e *msgJoinedError) Error() string {
	if isCalledFromRuntimePanic() {
		return fmt.Sprintf("% -+#.1v", Formatter{Error: e})
	}
	return e.msg
}

func (e *msgJoinedError) Format(s fmt.State, verb rune) {
	fmt.Fprintf(s, formatString(s, verb), Formatter{Error: e})
}

func (e msgJoinedError) MarshalJSON() ([]byte, error) {
	return marshalJSONError(&e)
}

func (e *msgJoinedError) Unwrap() []error {
	return e.errs
}

func (e *msgJoinedError) StackTrace() []uintptr {
	return e.stack
}

func (e *msgJoinedError) Details() map[string]interface{} {
	e.detailsMu.Lock()
	defer e.detailsMu.Unlock()

	if e.details == nil {
		e.details = make(map[string]interface{})
	}
	return e.details
}

func withStack(err error) E {
	e, ok := err.(E) //nolint:errorlint
	if ok {
		if len(e.StackTrace()) == 0 {
			return &noMsgError{
				err:       err,
				stack:     callers(1),
				details:   nil,
				detailsMu: new(sync.Mutex),
			}
		}
		return e
	}

	st := getExistingStackTrace(err)
	if len(st) == 0 {
		st = callers(1)
	}

	return &noMsgError{
		err:       err,
		stack:     st,
		details:   nil,
		detailsMu: new(sync.Mutex),
	}
}

// WithStack annotates err with a stack trace at the point WithStack was called,
// if err does not already have a stack trace.
// If err is nil, WithStack returns nil.
//
// Use WithStack instead of Wrap when you just want to convert an existing error
// into one with a stack trace. Use it as close to where the error originated
// as you can get.
//
// You can also use WithStack when you have an err which implements stackTracer
// interface but does not implement detailer interface as well, but you cannot
// provide initial details like you can with WithDetails.
//
// WithStack is similar to Errorf("%w", err), but returns err as-is if err
// already satisfies interface E and has a stack trace,
// and it returns nil if err is nil.
func WithStack(err error) E {
	if err == nil {
		return nil
	}

	return withStack(err)
}

// noMsgError wraps another error and has its
// own stack and but does not have its own msg.
type noMsgError struct {
	err       error
	stack     []uintptr
	details   map[string]interface{}
	detailsMu *sync.Mutex
}

func (e *noMsgError) Error() string {
	if isCalledFromRuntimePanic() {
		return fmt.Sprintf("% -+#.1v", Formatter{Error: e})
	}
	return e.err.Error()
}

func (e *noMsgError) Format(s fmt.State, verb rune) {
	fmt.Fprintf(s, formatString(s, verb), Formatter{Error: e})
}

func (e noMsgError) MarshalJSON() ([]byte, error) {
	return marshalJSONError(&e)
}

func (e *noMsgError) Unwrap() error {
	return e.err
}

func (e *noMsgError) StackTrace() []uintptr {
	return e.stack
}

func (e *noMsgError) Details() map[string]interface{} {
	e.detailsMu.Lock()
	defer e.detailsMu.Unlock()

	if e.details == nil {
		e.details = make(map[string]interface{})
	}
	return e.details
}

// Wrap returns an error annotating err with a stack trace
// at the point Wrap is called, and the supplied message.
// Wrapping is done even if err already has a stack trace.
// It records the original error as a cause.
// If err is nil, Wrap returns nil.
//
// Use Wrap when you want to make a new error with a different error message,
// while preserving the cause of the new error.
// If you want to reuse the err error message use WithMessage
// or Errorf instead.
func Wrap(err error, message string) E {
	if err == nil {
		return nil
	}

	return &causeError{
		err:       err,
		msg:       message,
		stack:     callers(0),
		details:   nil,
		detailsMu: new(sync.Mutex),
	}
}

// Wrapf returns an error annotating err with a stack trace
// at the point Wrapf is called, and the supplied message
// formatted according to a format specifier.
// Wrapping is done even if err already has a stack trace.
// It records the original error as a cause.
// It does not support %w format verb (use %s instead if you
// need to incorporate cause's error message).
// If err is nil, Wrapf returns nil.
//
// Use Wrapf when you want to make a new error with a different error message,
// preserving the cause of the new error.
// If you want to reuse the err error message use WithMessage
// or Errorf instead.
func Wrapf(err error, format string, args ...interface{}) E {
	if err == nil {
		return nil
	}

	return &causeError{
		err:       err,
		msg:       fmt.Sprintf(format, args...),
		stack:     callers(0),
		details:   nil,
		detailsMu: new(sync.Mutex),
	}
}

// causeError records another error as a causeError
// and has its own stack and msg.
type causeError struct {
	err       error
	msg       string
	stack     []uintptr
	details   map[string]interface{}
	detailsMu *sync.Mutex
}

func (e *causeError) Error() string {
	if isCalledFromRuntimePanic() {
		return fmt.Sprintf("% -+#.1v", Formatter{Error: e})
	}
	return e.msg
}

func (e *causeError) Format(s fmt.State, verb rune) {
	fmt.Fprintf(s, formatString(s, verb), Formatter{Error: e})
}

func (e causeError) MarshalJSON() ([]byte, error) {
	return marshalJSONError(&e)
}

func (e *causeError) Unwrap() error {
	return e.err
}

func (e *causeError) Cause() error {
	return e.err
}

func (e *causeError) StackTrace() []uintptr {
	return e.stack
}

func (e *causeError) Details() map[string]interface{} {
	e.detailsMu.Lock()
	defer e.detailsMu.Unlock()

	if e.details == nil {
		e.details = make(map[string]interface{})
	}
	return e.details
}

func withMessage(err error, prefix ...string) E {
	st := getExistingStackTrace(err)
	if len(st) == 0 {
		st = callers(1)
	}

	return &msgError{
		err:       err,
		msg:       prefixMessage(err.Error(), prefix...),
		stack:     st,
		details:   nil,
		detailsMu: new(sync.Mutex),
	}
}

// WithMessage annotates err with a prefix message or messages.
// If err does not have a stack trace, stack strace is recorded as well.
//
// It does not support controlling the delimiter. Use Errorf if you need that.
//
// If err is nil, WithMessage returns nil.
//
// WithMessage is similar to Errorf("%s: %w", prefix, err), but supports
// dynamic number of prefixes, and it returns nil if err is nil.
func WithMessage(err error, prefix ...string) E {
	if err == nil {
		return nil
	}

	return withMessage(err, prefix...)
}

// WithMessagef annotates err with a prefix message
// formatted according to a format specifier.
// If err does not have a stack trace, stack strace is recorded as well.
//
// It does not support %w format verb or controlling the delimiter.
// Use Errorf if you need that.
//
// If err is nil, WithMessagef returns nil.
//
// WithMessagef is similar to Errorf(format + ": %w", args..., err), but
// it returns nil if err is nil.
func WithMessagef(err error, format string, args ...interface{}) E {
	if err == nil {
		return nil
	}

	return withMessage(err, fmt.Sprintf(format, args...))
}

// Cause returns the result of calling the Cause method on err, if err's
// type contains a Cause method returning error.
// Otherwise, the err is unwrapped and the process is repeated.
// If unwrapping is not possible, Cause returns nil.
// Unwrapping stops if it encounters an error with
// Unwrap() method returning multiple errors.
func Cause(err error) error {
	for err != nil {
		c, ok := err.(causer)
		if ok {
			cause := c.Cause()
			if cause != nil {
				return cause //nolint:wrapcheck
			}
		}
		e, ok := err.(unwrapperJoined)
		if ok && len(e.Unwrap()) > 0 {
			return nil
		}
		err = Unwrap(err)
	}
	return err
}

// Unjoin returns the result of calling the Unwrap method on err, if err's
// type contains an Unwrap method returning multiple errors.
// Otherwise, the err is unwrapped and the process is repeated.
// If unwrapping is not possible, Unjoin returns nil.
// Unwrapping stops if it encounters an error with the Cause
// method returning error.
func Unjoin(err error) []error {
	for err != nil {
		e, ok := err.(unwrapperJoined)
		if ok {
			errs := e.Unwrap()
			if len(errs) > 0 {
				return errs
			}
		}
		c, ok := err.(causer)
		if ok && c.Cause() != nil {
			return nil
		}
		err = Unwrap(err)
	}
	return nil
}

// Details returns the result of calling the Details method on err,
// if err's type contains a Details method returning initialized map.
// Otherwise, the err is unwrapped and the process is repeated.
// If unwrapping is not possible, Details returns nil.
// Unwrapping stops if it encounters an error with the Cause
// method returning error, or Unwrap() method returning
// multiple errors.
//
// You can modify returned map to modify err's details.
func Details(err error) map[string]interface{} {
	for err != nil {
		dd := detailsOf(err)
		if dd != nil {
			return dd
		}
		c, ok := err.(causer)
		if ok && c.Cause() != nil {
			return nil
		}
		e, ok := err.(unwrapperJoined)
		if ok && len(e.Unwrap()) > 0 {
			return nil
		}
		err = Unwrap(err)
	}
	return nil
}

// Returns details of the err if it implements detailer interface.
// It does not unwrap and recurse.
func detailsOf(err error) map[string]interface{} {
	if err == nil {
		return nil
	}
	d, ok := err.(detailer)
	if ok {
		return d.Details()
	}
	return nil
}

// AllDetails returns a map build from calling the Details method on err
// and populating the map with key/value pairs which are not yet
// present. Afterwards, the err is unwrapped and the process is repeated.
// Unwrapping stops if it encounters an error with the Cause
// method returning error, or Unwrap() method returning
// multiple errors.
func AllDetails(err error) map[string]interface{} {
	res := make(map[string]interface{})
	for err != nil {
		for key, value := range detailsOf(err) {
			if _, ok := res[key]; !ok {
				res[key] = value
			}
		}
		c, ok := err.(causer)
		if ok && c.Cause() != nil {
			return res
		}
		e, ok := err.(unwrapperJoined)
		if ok && len(e.Unwrap()) > 0 {
			return res
		}
		err = Unwrap(err)
	}
	return res
}

// allDetailsUntilCauseOrJoined builds a map with details unwrapping errors
// until it hits a cause or joined errors, also returning it or them.
// This also means that it does not traverse errors returned by Join.
func allDetailsUntilCauseOrJoined(err error) (res map[string]interface{}, cause error, errs []error) { //nolint:revive,stylecheck,nonamedreturns
	res = make(map[string]interface{})
	cause = nil
	errs = nil

	for err != nil {
		for key, value := range detailsOf(err) {
			if _, ok := res[key]; !ok {
				res[key] = value
			}
		}
		c, ok := err.(causer)
		if ok {
			cause = c.Cause()
		}
		e, ok := err.(unwrapperJoined)
		if ok {
			errs = e.Unwrap()
		}
		if cause != nil || len(errs) > 0 {
			// It is possible that both cause and errs is set. One example is wrapError.
			return
		}
		err = Unwrap(err)
	}

	return
}

// causeOrJoined unwraps err repeatedly until it hits a cause or joined errors,
// returning it or them.
// This also means that it does not traverse errors returned by Join.
func causeOrJoined(err error) (cause error, errs []error) { //nolint:revive,stylecheck,nonamedreturns
	cause = nil
	errs = nil

	for err != nil {
		c, ok := err.(causer)
		if ok {
			cause = c.Cause()
		}
		e, ok := err.(unwrapperJoined)
		if ok {
			errs = e.Unwrap()
		}
		if cause != nil || len(errs) > 0 {
			// It is possible that both cause and errs is set. One example is wrapError.
			return
		}
		err = Unwrap(err)
	}

	return
}

// WithDetails wraps err into an error which implements the detailer interface
// to access a map with optional additional details about the error.
//
// If err does not have a stack trace, then this call is equivalent
// to calling WithStack, annotating err with a stack trace as well.
//
// Use WithDetails when you have an err which implements stackTracer interface
// but does not implement detailer interface as well. You can also use
// WithStack for that but you cannot provide initial details using WithStack
// like you can with WithDetails.
//
// It is also useful when err does implement detailer interface, but you want
// to reuse same err multiple times (e.g., pass same err to multiple
// goroutines), adding different details each time. Calling WithDetails
// always wraps err and adds an additional and independent layer of
// details on top of any existing details.
//
// You can provide initial details by providing pairs of keys (strings)
// and values (interface{}).
func WithDetails(err error, kv ...interface{}) E {
	if err == nil {
		return nil
	}

	if len(kv)%2 != 0 {
		panic(New("odd number of arguments for initial details"))
	}

	// We always initialize map because details were explicitly asked for.
	initMap := make(map[string]interface{})
	for i := 0; i < len(kv); i += 2 {
		key, ok := kv[i].(string)
		if !ok {
			panic(Errorf(`key "%v" must be a string, not %T`, kv[i], kv[i]))
		}
		initMap[key] = kv[i+1]
	}

	// Even if err is of type E, we still wrap it into another noMsgError error to
	// have another layer of details. This is where it is different from WithStack.
	// We do not have to check for type E explicitly because E implements stackTracer
	// so getExistingStackTrace returns its stack trace.
	st := getExistingStackTrace(err)
	if len(st) == 0 {
		st = callers(0)
	}

	return &noMsgError{
		err:       err,
		stack:     st,
		details:   initMap,
		detailsMu: new(sync.Mutex),
	}
}

// Join returns an error that wraps the given errors.
// Join also records the stack trace at the point it was called.
// Any nil error values are discarded.
// Join returns nil if errs contains no non-nil values.
// If there is only one non-nil value, Join behaves
// like WithStack on the non-nil value.
// The error formats as the concatenation of the strings obtained
// by calling the Error method of each element of errs, with a newline
// between each string.
//
// Join is similar to Errorf("%w\n%w\n", err1, err2), but supports
// dynamic number of errors, skips nil errors, and it returns
// the error as-is if there is only one non-nil error already
// with a stack trace.
func Join(errs ...error) E {
	nonNilErrs := make([]error, 0, len(errs))
	for _, err := range errs {
		if err != nil {
			nonNilErrs = append(nonNilErrs, err)
		}
	}

	if len(nonNilErrs) == 0 {
		return nil
	} else if len(nonNilErrs) == 1 {
		return withStack(nonNilErrs[0])
	}

	return &msgJoinedError{
		errs:      nonNilErrs,
		msg:       joinMessages(nonNilErrs),
		stack:     callers(0),
		details:   nil,
		detailsMu: new(sync.Mutex),
	}
}

// wrapError joins two errors (err and with), making err the cause of with.
type wrapError struct {
	err       error
	with      error
	stack     []uintptr
	details   map[string]interface{}
	detailsMu *sync.Mutex
}

func (e *wrapError) Error() string {
	if isCalledFromRuntimePanic() {
		return fmt.Sprintf("% -+#.1v", Formatter{Error: e})
	}
	return e.with.Error()
}

func (e *wrapError) Format(s fmt.State, verb rune) {
	fmt.Fprintf(s, formatString(s, verb), Formatter{Error: e})
}

func (e wrapError) MarshalJSON() ([]byte, error) {
	return marshalJSONError(&e)
}

func (e *wrapError) Unwrap() []error {
	return []error{e.with, e.err}
}

func (e *wrapError) Cause() error {
	return e.err
}

func (e *wrapError) StackTrace() []uintptr {
	return e.stack
}

func (e *wrapError) Details() map[string]interface{} {
	e.detailsMu.Lock()
	defer e.detailsMu.Unlock()

	if e.details == nil {
		e.details = make(map[string]interface{})
	}
	return e.details
}

// WrapWith makes the "err" error the cause of the "with" error.
// This is similar to Wrap but instead of using just an error
// message, you can provide a base error instead.
// If "err" is nil, WrapWith returns nil.
// If "with" is nil, WrapWith panics.
//
// If the "with" error does already have a stack trace,
// a stack trace is recorded at the point WrapWith was called.
//
// The new error wraps two errors, the "with" error and the "err" error,
// making it possible to use both Is and As on the new error to
// traverse both "with" and "err" errors at the same time.
//
// Note that the new error introduces a new context for details so
// any details from the "err" and "with" errors are not available
// through AllDetails on the new error.
//
// Use WrapWith when you want to make a new error using a base error
// with a different error message,
// while preserving the cause of the new error.
// If you want to reuse the "err" error message use Prefix or Errorf instead.
func WrapWith(err, with error) E {
	if with == nil {
		panic(New(`"with" argument cannot be nil`))
	}

	if err == nil {
		return nil
	}

	st := getExistingStackTrace(with)
	if len(st) == 0 {
		st = callers(0)
	}

	return &wrapError{
		err:       err,
		with:      with,
		stack:     st,
		details:   nil,
		detailsMu: new(sync.Mutex),
	}
}

// Prefix annotates err with a prefix message or messages of prefix errors,
// wrapping prefix errors at the same time.
// This is similar to WithMessage but instead of using just an error
// message, you can provide a base error instead.
// If err does not have a stack trace, stack strace is recorded as well.
//
// It does not support controlling the delimiter. Use Errorf if you need that.
//
// If err is nil, Prefix returns nil.
//
// Use Prefix when you want to make a new error using a base error or base errors
// and want to construct the new message through common prefixing.
// If you want to control how are messages combined, use Errorf.
// If you want to fully replace the message, use WrapWith.
//
// Prefix is similar to Errorf("%w: %w", prefixErr, err), but supports
// dynamic number of prefix errors, skips nil errors,
// does not record a stack trace if err already has it,
// and it returns nil if err is nil.
func Prefix(err error, prefix ...error) E {
	if err == nil {
		return nil
	}

	nonNilErrs := make([]error, 0, len(prefix))
	prefixes := make([]string, 0, len(prefix))
	for _, p := range prefix {
		if p != nil {
			nonNilErrs = append(nonNilErrs, p)
			prefixes = append(prefixes, p.Error())
		}
	}

	if len(nonNilErrs) == 0 {
		return withStack(err)
	}

	st := getExistingStackTrace(err)
	if len(st) == 0 {
		st = callers(0)
	}

	nonNilErrs = append(nonNilErrs, err)

	return &msgJoinedError{
		errs:      nonNilErrs,
		msg:       prefixMessage(err.Error(), prefixes...),
		stack:     st,
		details:   nil,
		detailsMu: new(sync.Mutex),
	}
}
