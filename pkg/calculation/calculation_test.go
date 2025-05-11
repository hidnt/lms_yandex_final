package calculation

import (
	"slices"
	"testing"

	"github.com/hidnt/lms_yandex_final/pkg/database"
)

type testCase struct {
	name        string
	expression  string
	id          int
	exceptedRes database.Expression
	wantError   bool
}

func TestCalc(t *testing.T) {
	testCases := []testCase{
		{
			name:       "correct expression",
			expression: "1012+123-24*10-4",
			id:         0,
			exceptedRes: database.Expression{
				Status: "under consideration",
				Result: 0,
				Actions: []database.Action{
					{
						Arg1:         1012,
						Arg2:         123,
						Result:       0,
						Operation:    "+",
						IdDepends:    []int64{-1, -1},
						Completed:    false,
						NowCalculate: false,
					},
					{
						Arg1:         24,
						Arg2:         10,
						Result:       0,
						Operation:    "*",
						IdDepends:    []int64{-1, -1},
						Completed:    false,
						NowCalculate: false,
					},
					{
						Arg1:         0,
						Arg2:         0,
						Result:       0,
						Operation:    "-",
						IdDepends:    []int64{1, 2},
						Completed:    false,
						NowCalculate: false,
					},
					{
						Arg1:         0,
						Arg2:         4,
						Result:       0,
						Operation:    "-",
						IdDepends:    []int64{3, -1},
						Completed:    false,
						NowCalculate: false,
					},
				},
			},
			wantError: false,
		},
		{
			name:       "correct expression",
			expression: "1+1",
			id:         0,
			exceptedRes: database.Expression{
				Status: "under consideration",
				Result: 0,
				Actions: []database.Action{
					{
						Arg1:         1,
						Arg2:         1,
						Result:       0,
						Operation:    "+",
						IdDepends:    []int64{-1, -1},
						Completed:    false,
						NowCalculate: false,
					},
				},
			},
			wantError: false,
		},
		{
			name:        "incorrect expression",
			expression:  "1238)",
			id:          0,
			exceptedRes: database.Expression{},
			wantError:   true,
		},
		{
			name:        "incorrect expression 2",
			expression:  "124+2-",
			id:          0,
			exceptedRes: database.Expression{},
			wantError:   true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ans, err := Calc(testCase.expression)
			if testCase.wantError {
				if err == nil {
					t.Fatalf("Excepted an err")
				}
			} else {
				if err != nil {
					t.Fatalf("Successful case is %s, but returns error: %s", testCase.expression, err.Error())
				}
				isEqual := true
				for i := 0; i < len(ans.Actions); i++ {
					if testCase.exceptedRes.Actions[i].ID != ans.Actions[i].ID ||
						testCase.exceptedRes.Actions[i].Arg1 != ans.Actions[i].Arg1 ||
						testCase.exceptedRes.Actions[i].Arg2 != ans.Actions[i].Arg2 ||
						testCase.exceptedRes.Actions[i].Result != ans.Actions[i].Result ||
						testCase.exceptedRes.Actions[i].Operation != ans.Actions[i].Operation ||
						testCase.exceptedRes.Actions[i].Completed != ans.Actions[i].Completed ||
						testCase.exceptedRes.Actions[i].NowCalculate != ans.Actions[i].NowCalculate ||
						!slices.Equal(testCase.exceptedRes.Actions[i].IdDepends, ans.Actions[i].IdDepends) {
						isEqual = false
						break
					}
				}
				if !isEqual {
					t.Fatalf("%v should be equal %v", ans.Actions, testCase.exceptedRes.Actions)
				}
			}
		})
	}
}
