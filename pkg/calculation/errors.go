package calculation

import "errors"

var (
	ErrUnknownOp        = errors.New("unknown operation")
	ErrIncorrectPriorOp = errors.New("incorrect prioritization operation")
	ErrNotEnoughtOp     = errors.New("not enough operations")
	ErrNotEnoughtNums   = errors.New("not enough nums")
	ErrDivByZero        = errors.New("division by zero")
	ErrCalc             = errors.New("calculation error")
)
