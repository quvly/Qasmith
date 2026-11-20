package diagnostic

import "fmt"

type Position struct {
	File         string
	Line, Column int
}

func (p Position) String() string {
	return fmt.Sprintf("%s:%d:%d", p.File, p.Line, p.Column)
}

type Error struct {
	Position      Position
	Code, Message string
	cause         error
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: error[%s]: %s", e.Position, e.Code, e.Message)
}

func At(p Position, code, format string, args ...any) error {
	return &Error{Position: p, Code: code, Message: fmt.Sprintf(format, args...)}
}

// Wrap preserves an underlying error for programmatic callers.
func Wrap(p Position, code string, cause error, format string, args ...any) error {
	return &Error{Position: p, Code: code, Message: fmt.Sprintf(format, args...) + ": " + cause.Error(), cause: cause}
}

func (e *Error) Unwrap() error { return e.cause }
