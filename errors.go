package ben

import (
	"errors"
	"fmt"
)

var (
	ErrEndItemSequence = errors.New("unexpected end of item sequence")

	ErrInputIsNotInteger   = InvalidInputError{"input is not an integer"}
	ErrInvalidStringLength = InvalidInputError{"invalid string length"}
	ErrUnknownTypeMarker   = InvalidInputError{"unknown type marker"}
	ErrCannotParseAsList   = InvalidInputError{"cannot parse as list"}
	ErrCannotParseAsDict   = InvalidInputError{"cannot parse as dictionary"}
	ErrKeyWithoutValue     = InvalidInputError{"key without value"}

	ErrNotAString   = ElementTypeError{"element is not a string"}
	ErrNotAnInteger = ElementTypeError{"element is not an integer"}
	ErrNotAList     = ElementTypeError{"element is not a list"}
	ErrNotADict     = ElementTypeError{"element is not a dictionary"}
	ErrNotImplBytes = ElementTypeError{"element does not implement Bytes()"}
)

type InvalidInputError struct {
	string
}

func (err InvalidInputError) Error() string {
	return fmt.Sprintf("Invalid Input Error: %s", err.string)
}

type ElementTypeError struct {
	string
}

func (t ElementTypeError) Error() string {
	return fmt.Sprintf("Type assertion error: %s", t.string)
}
