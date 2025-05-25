//go:build go1.21
// +build go1.21

package errors

import (
	stderrors "errors"
)

// ErrUnsupported indicates that a requested operation cannot be performed,
// because it is unsupported. For example, a call to [os.Link] when using a
// file system that does not support hard links.
//
// Functions and methods should not return this error but should instead
// return an error including appropriate context that satisfies
//
//	errors.Is(err, errors.ErrUnsupported)
//
// either by directly wrapping ErrUnsupported or by implementing an Is method.
//
// Functions and methods should document the cases in which an error
// wrapping this will be returned.
//
// This variable is the same as the standard errors.ErrUnsupported.
var ErrUnsupported = stderrors.ErrUnsupported
