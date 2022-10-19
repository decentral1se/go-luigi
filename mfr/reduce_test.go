// SPDX-FileCopyrightText: 2021 The Luigi Authors
//
// SPDX-License-Identifier: MIT

package mfr // import "github.com/ssbc/go-luigi/mfr"

import (
	"context"
	"fmt"
	"testing"

	"github.com/ssbc/go-luigi"
)

func ExampleReduceSink() {
	// adder sums all numbers in the stream
	adder := func(
		_ context.Context,
		acc, v interface{},
	) (interface{}, error) {
		if acc == nil {
			acc = 0
		}
		sum := v.(int) + acc.(int)
		return interface{}(sum), nil
	}
	reducer := NewReduceSink(adder)

	numbers := luigi.SliceSource([]interface{}{0, 1, 2, 3, 4})
	_ = luigi.Pump(context.Background(), reducer, &numbers)

	total, _ := reducer.Value()
	fmt.Println(total)
	// Output: 10
}

type TypeError struct {
	expected, got interface{}
}

func (err TypeError) Error() string {
	return fmt.Sprintf("expected type %T, got %T", err.expected, err.got)
}

func TestReduce(t_ *testing.T) {
	type testcase struct {
		in             []interface{}
		results        []interface{}
		pourErrStrings []string
	}

	mkTest := func(tc testcase) func(*testing.T) {
		return func(t *testing.T) {
			rs := NewReduceSink(func(ctx context.Context, acc, v interface{}) (interface{}, error) {
				if acc == nil {
					return v, nil
				}

				f64Acc, ok := acc.(float64)
				if !ok {
					return nil, TypeError{expected: f64Acc, got: acc}
				}

				f64V, ok := v.(float64)
				if !ok {
					return nil, TypeError{expected: f64V, got: v}
				}

				return f64Acc*0.75 + f64V*0.25, nil
			})

			var iCheck int
			var check luigi.FuncSink = func(ctx context.Context, v interface{}, err error) error {
				defer func() { iCheck++ }()
				if err != nil && iCheck < len(tc.results) {
					return fmt.Errorf("received close, but there are values left (i:%v, v:%v, len(results):%v",
						iCheck, v, len(tc.results))
				} else if err != nil {
					return nil
				} else if iCheck >= len(tc.results) {
					return fmt.Errorf("received more values than expected (i:%v, v:%v, len(results):%v",
						iCheck, v, len(tc.results))
				}

				if v != tc.results[iCheck] {
					return fmt.Errorf("expected value %v, but got %v (i=%v)", tc.results[iCheck], v, iCheck)
				}

				return nil
			}

			regCancel := rs.Register(check)
			defer regCancel()

			ctx, ctxCancel := context.WithCancel(context.Background())
			defer ctxCancel()

			for i, v := range tc.in {
				err := rs.Pour(ctx, v)
				if err == nil && tc.pourErrStrings[i] != "" {
					t.Errorf("expected pour error %q but got: %v", tc.pourErrStrings[i], err)
				} else if err != nil && tc.pourErrStrings[i] == "" {
					t.Errorf("expected no pour error but got: %v", err)
				}
			}
		}
	}

	tcs := []testcase{
		{
			in:             []interface{}{1.0, 2.0, 3.0},
			results:        []interface{}{1.0, 3.0/4 + 1.0/2, (1.0 + 3.0/4 + 1.0/2) * 3 / 4},
			pourErrStrings: []string{"", "", ""},
		},
	}

	for i, tc := range tcs {
		t_.Run(fmt.Sprint(i), mkTest(tc))
	}
}
