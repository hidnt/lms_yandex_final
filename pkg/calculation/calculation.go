package calculation

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hidnt/lms_yandex_final/pkg/database"
)

func Calc(expression string) (database.Expression, error) {
	var parts []string
	var curPart string
	var lastIsNotNumber bool = false
	for _, char := range expression {
		if char == ' ' {
			continue
		}
		if char == '-' && (curPart == "" || curPart == "(") {
			curPart += string(char)
			continue
		}
		if strings.ContainsAny(string(char), "0123456789.") {
			if lastIsNotNumber && curPart == "-" {
				parts = append(parts, curPart)
				curPart = ""
				lastIsNotNumber = false
			}
			curPart += string(char)
			continue
		}
		if curPart != "" {
			if curPart == "-" {
				lastIsNotNumber = false
			} else {
				lastIsNotNumber = true
			}
			parts = append(parts, curPart)
			curPart = ""
		}
		if strings.ContainsAny(string(char), "+*/()") {
			parts = append(parts, string(char))
			lastIsNotNumber = false
			if string(char) == ")" {
				lastIsNotNumber = true
			}
			continue
		}
		if string(char) == "-" {
			curPart += "-"
			continue
		}
		return database.Expression{}, ErrUnknownOp
	}

	if curPart != "" {
		parts = append(parts, curPart)
	}

	var nums []string
	var operators []string
	var actions []database.Action
	curAction := 1

	priority := map[string]int{
		"+": 1,
		"-": 1,
		"*": 2,
		"/": 2,
		"(": 0,
	}

	calculate := func() error {
		if len(operators) == 0 {
			return ErrNotEnoughtOp
		}
		if len(nums) < 2 && len(operators) >= 1 {
			err := ErrNotEnoughtNums
			return err
		}

		depends := []int64{}
		var left, right float64

		if num, err := strconv.ParseFloat(nums[len(nums)-2], 64); err == nil {
			left = num
			depends = append(depends, -1)
		} else {
			left = 0
			n, err := strconv.ParseInt(nums[len(nums)-2][1:], 10, 64)
			if err != nil {
				return err
			}
			depends = append(depends, n)
		}
		if num, err := strconv.ParseFloat(nums[len(nums)-1], 64); err == nil {
			right = num
			depends = append(depends, -1)
		} else {
			right = 0
			n, err := strconv.ParseInt(nums[len(nums)-1][1:], 10, 64)
			if err != nil {
				return err
			}
			depends = append(depends, n)
		}
		nums = nums[:len(nums)-2]

		operator := operators[len(operators)-1]
		operators = operators[:len(operators)-1]

		actions = append(actions, database.Action{
			Arg1:         left,
			Arg2:         right,
			Result:       0,
			Operation:    operator,
			IdDepends:    depends,
			Completed:    false,
			NowCalculate: false,
		})

		nums = append(nums, fmt.Sprintf("d%d", curAction))
		curAction++

		return nil
	}

	for _, part := range parts {
		if _, err := strconv.ParseFloat(part, 64); err == nil {
			nums = append(nums, part)
			continue
		}
		if part == "(" {
			operators = append(operators, part)
			continue
		}
		if part == ")" {
			for len(operators) > 0 && operators[len(operators)-1] != "(" {
				if err := calculate(); err != nil {
					return database.Expression{}, err
				}
			}
			if len(operators) == 0 {
				return database.Expression{}, ErrIncorrectPriorOp
			}
			operators = operators[:len(operators)-1]
		} else {
			for len(operators) > 0 && priority[operators[len(operators)-1]] >= priority[part] {
				if err := calculate(); err != nil {
					return database.Expression{}, err
				}
			}
			operators = append(operators, part)
		}
	}

	for len(operators) > 0 {
		if err := calculate(); err != nil {
			return database.Expression{}, err
		}
	}

	if len(nums) != 1 {
		return database.Expression{}, ErrCalc
	}

	if len(nums) == 1 && len(actions) == 0 {
		if n, err := strconv.ParseFloat(nums[0], 64); err == nil {
			actions = append(actions, database.Action{
				Arg1:         n,
				Arg2:         0,
				Result:       0,
				Operation:    "+",
				IdDepends:    []int64{-1, -1},
				Completed:    false,
				NowCalculate: false,
			})
		}
	}

	exprs := database.Expression{
		Status:  "under consideration",
		Result:  0,
		Actions: actions,
	}

	return exprs, nil
}
