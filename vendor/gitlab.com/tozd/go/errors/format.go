package errors

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// Copied from fmt/print.go.
const (
	percentBangString = "%!"
	nilAngleString    = "<nil>"
	badPrecString     = "%!(BADPREC)"
)

const (
	stackTraceHelp     = "stack trace (most recent call first):\n"
	multipleErrorsHelp = "the above error joins errors:\n"
	causeHelp          = "the above error was caused by the following error:\n"
)

// Similar to one in fmt/print.go.
func badVerb(s fmt.State, verb rune, arg interface{}) {
	_, _ = io.WriteString(s, percentBangString)
	_, _ = io.WriteString(s, string([]rune{verb}))
	_, _ = io.WriteString(s, "(")
	if arg != nil {
		_, _ = io.WriteString(s, reflect.TypeOf(arg).String())
		_, _ = io.WriteString(s, "=")
		fmt.Fprintf(s, "%v", arg)
	} else {
		_, _ = io.WriteString(s, nilAngleString)
	}
	_, _ = io.WriteString(s, ")")
}

// Copied from zerolog/console.go.
func needsQuote(s string) bool {
	for i := range s {
		if s[i] < 0x20 || s[i] > 0x7e || s[i] == ' ' || s[i] == '\\' || s[i] == '"' {
			return true
		}
	}
	return false
}

func writeLinesPrefixed(w io.Writer, linePrefix, s string) {
	lines := strings.Split(s, "\n")
	// Trim empty lines at start.
	for len(lines) > 0 && lines[0] == "" {
		lines = lines[1:]
	}
	// Trim empty lines at end.
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	for _, line := range lines {
		_, _ = io.WriteString(w, linePrefix)
		_, _ = io.WriteString(w, line)
		_, _ = io.WriteString(w, "\n")
	}
}

func useKnownInterface(err error) bool {
	for err != nil {
		switch err.(type) { //nolint:errorlint
		case stackTracer, pkgStackTracer, goErrorsStackTracer, erisStackTracer, detailer, placeholderStackTracer:
			// We return true only on interfaces with data. Not on causer.
			// We do not care if they really do have data at this point though.
			return true
		}
		// We stop unwrapping if we hit cause or join.
		c, ok := err.(causer)
		if ok && c.Cause() != nil {
			return false
		}
		e, ok := err.(unwrapperJoined)
		if ok && len(e.Unwrap()) > 0 {
			return false
		}
		err = Unwrap(err)
	}
	return false
}

func useFormatter(err error) bool {
	if useKnownInterface(err) {
		return false
	}

	// We check for this interface without unwrapping because it does not work with wrapping anyway.
	_, ok := err.(fmt.Formatter)
	return ok
}

func isForeignFormatter(err error) bool {
	// Our errors implement fmt.Formatter but we want to return false for them because
	// they just call into our Formatter which would lead to infinite recursion.
	switch err.(type) { //nolint:errorlint
	case *fundamentalError, *msgError, *msgJoinedError, *noMsgError, *causeError, *wrapError,
		*placeholderError, *placeholderCauseError, *placeholderJoinedError, *placeholderJoinedCauseError:
		return false
	}

	// We check for this interface without unwrapping because it does not work with wrapping anyway.
	_, ok := err.(fmt.Formatter)
	return ok
}

func (f Formatter) formatError(s fmt.State, w io.Writer, indent int, err error) {
	linePrefix := ""
	if indent > 0 {
		width, ok := s.Width()
		if ok {
			linePrefix = strings.Repeat(strings.Repeat(" ", width), indent)
		} else {
			linePrefix = strings.Repeat("\t", indent)
		}
	}

	var cause error
	var errs []error
	precision, ok := s.Precision()
	if !ok {
		// We explicitly set it to 0.
		// See: https://github.com/golang/go/issues/61913
		precision = 0
	}

	if precision >= 2 && isForeignFormatter(err) || err == nil {
		writeLinesPrefixed(w, linePrefix, fmt.Sprintf(formatString(s, 'v'), err))
		// Here we return because we assume formatting does recurse itself or at least
		// the user requested us to not recuse if the error implements fmt.Formatter.
		return
	}

	if useFormatter(err) {
		writeLinesPrefixed(w, linePrefix, fmt.Sprintf(formatString(s, 'v'), err))
		// Here we still recurse ourselves because we assume formatting just formats the error and
		// does not recurse if it does not implement those interfaces which we checked in useFormatter.
		if precision == 1 || precision == 3 {
			cause, errs = causeOrJoined(err)
		}
	} else {
		f.formatMsg(w, linePrefix, err)
		var details map[string]interface{}
		if s.Flag('#') {
			details, cause, errs = allDetailsUntilCauseOrJoined(err)
		} else if precision == 1 || precision == 3 {
			cause, errs = causeOrJoined(err)
		}
		if s.Flag('#') {
			f.formatDetails(w, linePrefix, details)
		}
		if s.Flag('+') {
			f.formatStack(s, w, linePrefix, err)
		}
	}

	if precision == 1 || precision == 3 { //nolint:nestif
		buf := new(bytes.Buffer)

		// It is possible that both cause and errs is set. In that case we first
		// recurse into errs and then into the cause, so that it is clear which
		// "above error" joins the errors (not the cause). Because cause is not
		// indented it is hopefully clearer that its "above error" does not mean
		// the last error among joined but the one higher up before indentation.

		if len(errs) > 0 {
			first := true
			for _, er := range errs {
				// er should never be nil, but we still check.
				// We also make sure we do not repeat cause here or repeat an error without any additional information.
				if er != nil && er != cause && !isSubsumedError(err, er) { //nolint:errorlint,err113
					// We format error to the buffer so that we can see if anything was written.
					buf.Reset()
					f.formatError(s, buf, indent+1, er)
					// If nothing was written, we skip this error.
					if buf.Len() == 0 {
						continue
					}

					if first {
						first = false
						if s.Flag('-') {
							if s.Flag(' ') {
								_, _ = io.WriteString(w, "\n")
							}
							writeLinesPrefixed(w, linePrefix, multipleErrorsHelp)
						}
					}
					if s.Flag(' ') {
						_, _ = io.WriteString(w, "\n")
					}
					_, _ = io.Copy(w, buf)
				}
			}
		}

		if cause != nil {
			// We format error to the buffer so that we can see if anything was written.
			buf.Reset()
			f.formatError(s, buf, indent, cause)
			// Only if something was written we continue.
			if buf.Len() > 0 {
				if s.Flag('-') {
					if s.Flag(' ') {
						_, _ = io.WriteString(w, "\n")
					}
					writeLinesPrefixed(w, linePrefix, causeHelp)
				}
				if s.Flag(' ') {
					_, _ = io.WriteString(w, "\n")
				}
				_, _ = io.Copy(w, buf)
			}
		}
	}
}

func (f Formatter) formatMsg(w io.Writer, linePrefix string, err error) {
	getMessage := f.GetMessage
	if getMessage == nil {
		getMessage = defaultGetMessage
	}

	writeLinesPrefixed(w, linePrefix, getMessage(err))
}

// Similar to writeFields in zerolog/console.go.
func (f Formatter) formatDetails(w io.Writer, linePrefix string, details map[string]interface{}) {
	fields := make([]string, len(details))
	i := 0
	for field := range details {
		fields[i] = field
		i++
	}
	sort.Strings(fields)
	for _, field := range fields {
		value := details[field]
		var v string
		switch tValue := value.(type) {
		case string:
			if needsQuote(tValue) {
				v = strconv.Quote(tValue)
			} else {
				v = tValue
			}
		case json.Number:
			v = string(tValue)
		default:
			b, err := marshalWithoutEscapeHTML(tValue)
			if err != nil {
				v = fmt.Sprintf("[error: %v]", err)
			} else {
				v = string(b)
			}
		}
		writeLinesPrefixed(w, linePrefix, fmt.Sprintf("%s=%s\n", field, v))
	}
}

func (f Formatter) formatStack(s fmt.State, w io.Writer, linePrefix string, err error) {
	var stToFormat interface{}
	st := getExistingStackTrace(err)
	if len(st) > 0 {
		stToFormat = StackFormatter{st}
	} else {
		placeholderErr, ok := err.(placeholderStackTracer)
		if !ok {
			return
		}
		placeholderSt := placeholderErr.StackTrace()
		if len(placeholderSt) == 0 {
			return
		}
		stToFormat = placeholderSt
	}

	if s.Flag('-') {
		writeLinesPrefixed(w, linePrefix, stackTraceHelp)
	}
	var result string
	width, ok := s.Width()
	if ok {
		result = fmt.Sprintf("%+*v", width, stToFormat)
	} else {
		result = fmt.Sprintf("%+v", stToFormat)
	}
	writeLinesPrefixed(w, linePrefix, result)
}

func defaultGetMessage(err error) string {
	return err.Error()
}

// Formatter formats an error as text and marshals the error as JSON.
type Formatter struct {
	Error error

	// Provide a function to obtain the error's message.
	// By default error's Error() is called.
	GetMessage func(error) string `exhaustruct:"optional"`
}

// Format formats the error as text according to the fmt.Formatter interface.
//
// The error does not have to necessary come from this package and it will be formatted
// in the same way if it implements interfaces used by this package (e.g., stackTracer
// or detailer interfaces). By default, only if those interfaces are not implemented,
// but fmt.Formatter interface is, formatting will be delegated to the error itself.
// You can change this default through format precision.
//
// Errors which do come from this package can be directly formatted by the fmt package
// in the same way as this function does as they implement fmt.Formatter interface.
// If you are not sure about the source of the error, it is safe to call this function
// on them as well.
//
// The following verbs are supported:
//
//	%s    the error message
//	%q    the quoted error message
//	%v    by default the same as %s
//
// You can control how is %v formatted through the width and precision arguments and
// flags. The width argument controls the width of the indent step in spaces. The default
// (no width argument) indents with a tab step.
// Width is passed through to the stack trace formatting.
//
// The following flags for %v are supported:
//
//	'#'   list details as key=value lines after the error message, when available
//	'+'   follow with the %+v formatted stack trace, if available
//	'-'   add human friendly messages to delimit parts of the text
//	' '   add extra newlines to separate parts of the text better
//
// Precision is specified by a period followed by a decimal number and enable
// modes of operation. The following modes are supported:
//
//	.0    do not change default behavior, this is the default
//	.1    recurse into error causes and joined errors
//	.2    prefer error's fmt.Formatter interface implementation if error implements it
//	.3    recurse into error causes and joined errors, but prefer fmt.Formatter
//	      interface implementation if any error implements it; this means that
//	      recursion stops if error's formatter does not recurse
//
// When any flag or non-zero precision mode is used, it is assured that the text
// ends with a newline, if it does not already do so.
func (f Formatter) Format(s fmt.State, verb rune) {
	getMessage := f.GetMessage
	if getMessage == nil {
		getMessage = defaultGetMessage
	}

	switch verb {
	case 'v':
		precision, ok := s.Precision()
		if !ok {
			// We explicitly set it to 0.
			// See: https://github.com/golang/go/issues/61913
			precision = 0
		}
		if precision < 0 || precision > 3 {
			_, _ = io.WriteString(s, badPrecString)
			break
		}
		if s.Flag('#') || s.Flag('+') || s.Flag('-') || s.Flag(' ') || precision > 0 {
			f.formatError(s, s, 0, f.Error)
			break
		}
		fallthrough
	case 's':
		if f.Error != nil {
			_, _ = io.WriteString(s, getMessage(f.Error))
		} else {
			fmt.Fprintf(s, "%s", f.Error)
		}
	case 'q':
		if f.Error != nil {
			fmt.Fprintf(s, "%q", getMessage(f.Error))
		} else {
			fmt.Fprintf(s, "%q", f.Error)
		}
	default:
		badVerb(s, verb, f.Error)
	}
}
