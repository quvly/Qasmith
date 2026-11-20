package extension

import (
	"strconv"
	"strings"
)

// ArrayType constructs a fixed-size array type whose outer dimension is length.
func ArrayType(element Type, length int) Type {
	return element + Type("["+strconv.Itoa(length)+"]")
}

// Array reports the element type and outermost length of an array.
func (t Type) Array() (element Type, length int, ok bool) {
	s := string(t)
	i := strings.LastIndexByte(s, '[')
	if i <= 0 || !strings.HasSuffix(s, "]") {
		return "", 0, false
	}
	length, err := strconv.Atoi(s[i+1 : len(s)-1])
	return Type(s[:i]), length, err == nil && length > 0
}

// Classical reports whether a type is a scalar or fixed-size array of classical values.
func (t Type) Classical() bool {
	for {
		switch t {
		case Int, Float, Bool:
			return true
		}
		element, _, ok := t.Array()
		if !ok {
			return false
		}
		t = element
	}
}
