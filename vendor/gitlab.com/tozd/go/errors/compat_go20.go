//go:build go1.20
// +build go1.20

package errors

import (
	stderrors "errors"
	"fmt"
)

var formatString = fmt.FormatString //nolint:gochecknoglobals

var (
	stderrorsIs = stderrors.Is //nolint:gochecknoglobals
	stderrorsAs = stderrors.As //nolint:gochecknoglobals
)
