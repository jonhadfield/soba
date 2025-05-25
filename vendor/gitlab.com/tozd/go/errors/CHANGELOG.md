# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.10.0] - 2024-09-09

### Added

- Expose all error's data when using in an unhandled panic.
  [#16](https://gitlab.com/tozd/go/errors/-/issues/5)

## [0.9.0] - 2024-09-06

### Added

- Support for Go 1.22 and 1.23.

### Removed

- Support for Go 1.16.

## [0.8.1] - 2023-12-04

### Fixed

- JSON marshal of value receiver errors.
  [#15](https://gitlab.com/tozd/go/errors/-/issues/15)

## [0.8.0] - 2023-10-12

### Added

- Detect stack traces in errors made by `github.com/rotisserie/eris` package.
- `GetMessage` to `Formatter` to control how error messages are obtain from
  errors during formatting.

## [0.7.2] - 2023-09-21

### Fixed

- Deep compare stack traces when formatting.

## [0.7.1] - 2023-09-20

### Fixed

- Use mutex when initializing details to prevent race conditions.
- Access details during formatting only when they are needed.

## [0.7.0] - 2023-09-19

### Added

- `Prefix` function to construct a new error by prefixing an error with base errors.

## [0.6.0] - 2023-09-16

### Added

- `WithDetails` now accepts optional pairs of keys and values as initial details.
- `Errorf` supports multiple `%w` when used with Go 1.20 or newer.
- `StackFormatter` which allows you to format stacks and marshal them to JSON.
- `Formatter` which allows you to format errors (including ones not from this package)
  and marshal them to JSON.
- Error formatting accepts multiple flags to control the output, including `#`
  flag to enable formatting of any details available on an error. It also accepts
  width to control indentation and precision to control if formatting recurses and/or
  uses error's `fmt.Formatter` implementation.
- Support for Go 1.21.
- `StackTrace` type alias for better compatibility with `github.com/pkg/errors`.
- `Unjoin` function to find joined errors.
- `UnmarshalJSON` to unmarshal JSON errors into placeholder errors which can then
  be formatted in the same way as other errors from this package.
- `WithWrap` function to set an error as a cause of the base error.

### Changed

- Formatting of errors has been changed so that `%+v` resembles
  the formatting of `github.com/pkg/errors` while additional formatting flags
  are necessary to obtain more verbose formatting previously done by this
  package (e.g., `-` for human-friendly messages to delimit parts of the text,
  ' ' to add extra newlines to separate parts of the text better). You can
  replace all `%+v` in your code with `% +-.3v` to obtain previous verbose formatting
  and `% #+-.3v` if you want to include new support for formatting details.
  [#5](https://gitlab.com/tozd/go/errors/-/issues/5)
  [#8](https://gitlab.com/tozd/go/errors/-/issues/8)
- Error formatting now by default uses `fmt.Formatter` implementation of an error
  only if the error does not implement interfaces used by this package (e.g.,
  `stackTracer` or `detailer`). This is to assure consistent error formatting
  when possible. You can change this default through format precision.
- `Details` now unwraps the error to find the first one providing
  details, only until an error with a cause or which
  wraps multiple errors.
- `AllDetails` collect details only until an error with a cause or which
  wraps multiple errors.
- JSON marshaling adds fields from error's details into JSON.
  [#7](https://gitlab.com/tozd/go/errors/-/issues/7)
- `Wrap` and `Wrapf` return `nil` if provided error is `nil`.

### Removed

- Remove `StackFormat` and `StackMarshalJSON` in favor of `StackFormatter`.

## [0.5.0] - 2023-06-06

### Added

- `StackFormat` to format the provided stack trace as text.
- `StackMarshalJSON` marshals the provided stack trace as JSON.
- Support wrapping multiple errors with `errors.Join`.
  [#4](https://gitlab.com/tozd/go/errors/-/issues/4)

### Changed

- `Wrap` behaves like `New` and `Wrapf` like `Errorf` if provided error is nil
  instead of returning `nil`.
  [#2](https://gitlab.com/tozd/go/errors/-/issues/2)
- Package is tested only on Go 1.16 and newer.
- Lines `stack trace (most recent call first):` and
  `the above error was caused by the following error:` changed to lower case.

## [0.4.1] - 2022-04-21

### Fixed

- Initialize details when calling `WithDetails` to prevent race conditions.

## [0.4.0] - 2022-04-20

### Added

- Errors returned by this package provide also optional details map accessible
  through `detailer` interface.
- `WithDetails` which wraps an error exposing access to (a potentially new layer of)
  details about the error.

## [0.3.0] - 2022-01-03

### Changed

- Change license to Apache 2.0.

## [0.2.0] - 2021-12-01

### Changed

- `errors.Cause` handles better `Cause` which returns `nil`.
- JSON marshaling of foreign errors uses `errors.Cause`.

## [0.1.0] - 2021-11-30

### Added

- First public release.

[unreleased]: https://gitlab.com/tozd/go/errors/-/compare/v0.10.0...main
[0.10.0]: https://gitlab.com/tozd/go/errors/-/compare/v0.9.0...v0.10.0
[0.9.0]: https://gitlab.com/tozd/go/errors/-/compare/v0.8.1...v0.9.0
[0.8.1]: https://gitlab.com/tozd/go/errors/-/compare/v0.8.0...v0.8.1
[0.8.0]: https://gitlab.com/tozd/go/errors/-/compare/v0.7.2...v0.8.0
[0.7.2]: https://gitlab.com/tozd/go/errors/-/compare/v0.7.1...v0.7.2
[0.7.1]: https://gitlab.com/tozd/go/errors/-/compare/v0.7.0...v0.7.1
[0.7.0]: https://gitlab.com/tozd/go/errors/-/compare/v0.6.0...v0.7.0
[0.6.0]: https://gitlab.com/tozd/go/errors/-/compare/v0.5.0...v0.6.0
[0.5.0]: https://gitlab.com/tozd/go/errors/-/compare/v0.4.1...v0.5.0
[0.4.1]: https://gitlab.com/tozd/go/errors/-/compare/v0.4.0...v0.4.1
[0.4.0]: https://gitlab.com/tozd/go/errors/-/compare/v0.3.0...v0.4.0
[0.3.0]: https://gitlab.com/tozd/go/errors/-/compare/v0.2.0...v0.3.0
[0.2.0]: https://gitlab.com/tozd/go/errors/-/compare/v0.1.0...v0.2.0
[0.1.0]: https://gitlab.com/tozd/go/errors/-/tags/v0.1.0

<!-- markdownlint-disable-file MD024 -->
