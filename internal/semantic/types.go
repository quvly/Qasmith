package semantic

import (
	"strconv"
	"strings"

	"github.com/quvly/Qasmith/pkg/extension"
)

const (
	Bool     = extension.Bool
	Register = Type("QReg")
)

func RegisterType(size int) Type { return Type("QReg<" + strconv.Itoa(size) + ">") }

func RegisterSize(t Type) (int, bool) {
	if t == Register {
		return 0, true
	}
	size, prefix := strings.CutPrefix(string(t), "QReg<")
	size, suffix := strings.CutSuffix(size, ">")
	if !prefix || !suffix {
		return 0, false
	}
	n, err := strconv.Atoi(size)
	return n, err == nil && n > 0
}

func quantum(t Type) bool {
	_, reg := RegisterSize(t)
	return reg || t == Qubit
}

func annotation(t Type) (Type, bool) {
	if n, ok := RegisterSize(t); ok && n > 0 {
		return RegisterType(n), true
	}
	if t == Qubit || t == Void || t == MeasuredResult {
		return t, true
	}
	if !t.Classical() {
		return "", false
	}
	// Normalize lengths with leading zeroes from source annotations.
	var dimensions []int
	for {
		elem, n, ok := t.Array()
		if !ok {
			break
		}
		dimensions = append(dimensions, n)
		t = elem
	}
	for i := len(dimensions) - 1; i >= 0; i-- {
		t = extension.ArrayType(t, dimensions[i])
	}
	return t, true
}

func arrayFits(t Type, remaining int) bool {
	for {
		element, n, ok := t.Array()
		if !ok {
			return true
		}
		if n > remaining {
			return false
		}
		remaining = (remaining - n) / n
		t = element
	}
}

func assignable(from, to Type) bool {
	if from == to || from == Int && to == Float {
		return true
	}
	n, reg := RegisterSize(to)
	return from == Register && reg && n > 0
}
