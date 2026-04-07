package notex

import (
	"reflect"
	"testing"
)

func TestMergeInt64Dedupe(t *testing.T) {
	cases := []struct {
		a, b []int64
		want []int64
	}{
		{nil, nil, nil},
		{[]int64{1, 2}, nil, []int64{1, 2}},
		{nil, []int64{3}, []int64{3}},
		{[]int64{1, 2}, []int64{2, 3}, []int64{1, 2, 3}},
		{[]int64{1}, []int64{1}, []int64{1}},
		{[]int64{0, -1, 5}, []int64{5, 6}, []int64{0, -1, 5, 6}},
	}
	for _, tc := range cases {
		got := mergeInt64Dedupe(tc.a, tc.b)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("mergeInt64Dedupe(%v, %v) = %v want %v", tc.a, tc.b, got, tc.want)
		}
	}
}
