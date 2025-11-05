package types

import (
	"reflect"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
)

func TestStringSlice(t *testing.T) {
	type S string
	type V string
	tests := []struct {
		name string
		s    []S
		want []V
	}{
		{
			name: "empty slice",
			s:    []S{},
			want: []V{},
		},
		{
			name: "one element",
			s:    []S{"a"},
			want: []V{"a"},
		},
		{
			name: "nil val",
			s:    nil,
			want: []V{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StringSlice[V](tt.s)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSumUint8(t *testing.T) {
	tests := []struct {
		name string
		args []uint8
		want uint8
	}{
		{
			name: "type can hold sum",
			args: []uint8{10, 0x01},
			want: 11,
		},
		{
			name: "type can not hold sum",
			args: []uint8{2, 0xff}, // 2 + 255 = 1
			want: 1,
		},
		{
			name: "type can not hold intermediate sum",
			args: []uint8{1, 0xff, 100}, // 1 + 255 + 100 = 100
			want: 100,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := Sum(tt.args)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSumInt8(t *testing.T) {
	tests := []struct {
		name string
		args []int8
		want int8
	}{
		{
			name: "type can hold sum",
			args: []int8{10, 0x01},
			want: 11,
		},
		{
			name: "type can not hold sum - positive",
			args: []int8{1, 0x7f}, // 2 + 255 = 1
			want: -128,
		},
		{
			name: "type can not hold sum - negative",
			args: []int8{-1, -0x80}, // -128 + -1 = 127
			want: 127,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := Sum(tt.args)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSliceClone(t *testing.T) {
	tests := []struct {
		name  string
		slice []string
	}{
		{
			"string slice:non empty",
			[]string{"one", "two"},
		},
		{
			"string slice:empty",
			[]string{},
		},
		{
			"string slice:nil",
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SliceClone(tt.slice)
			assert.ElementsMatch(t, tt.slice, got)
			if len(tt.slice) > 0 {
				assert.False(t, unsafe.Pointer(&tt.slice[0]) == unsafe.Pointer(&got[0]))
			}
		})
	}
}

func Test_PointerSliceINT(t *testing.T) {
	tests := []struct {
		name  string
		slice []int
		want  []*int
	}{{
		"int",
		[]int{1, 2, 3, 4},
		[]*int{Ptr(1), Ptr(2), Ptr(3), Ptr(4)},
	},
		{
			"nil",
			nil,
			[]*int{},
		},
		{
			"zero length",
			[]int{},
			[]*int{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PointerSlice(tt.slice); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReduce_WithMap(t *testing.T) {
	type args struct {
		list []int
		acc  map[int]int
		f    func(acc map[int]int, v int) map[int]int
	}
	tests := []struct {
		name string
		args args
		want map[int]int
	}{
		{
			name: "reduce - count",
			args: args{
				list: []int{1, 2, 3, 4, 4, 4, 56, 2, 3},
				acc:  map[int]int{},
				f: func(acc map[int]int, v int) map[int]int {
					acc[v]++
					return acc
				},
			},
			want: map[int]int{1: 1, 2: 2, 3: 2, 4: 3, 56: 1},
		},
		{
			name: "reduce - sum",
			args: args{
				list: []int{1, 2, 3, 4, 4, 4, 56, 2, 3},
				acc:  map[int]int{},
				f: func(acc map[int]int, v int) map[int]int {
					acc[v] += v
					return acc
				},
			},
			want: map[int]int{1: 1, 2: 4, 3: 6, 4: 12, 56: 56},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Reduce(tt.args.list, tt.args.acc, tt.args.f); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Reduce() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReduce_ToInt(t *testing.T) {
	// table driven test
	type args struct {
		list []int
		acc  int
		f    func(acc int, v int) int
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "reduce - sum",
			args: args{
				list: []int{1, 2, 3, 4, 4, 4, 56, 2, 3},
				acc:  0,
				f: func(acc int, v int) int {
					return acc + v
				},
			},
			want: 79,
		},
		{
			name: "reduce - product",
			args: args{
				list: []int{1, 2, 3, 4, 4, 2, 3},
				acc:  1,
				f: func(acc int, v int) int {
					return acc * v
				},
			},
			want: 576,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Reduce(tt.args.list, tt.args.acc, tt.args.f); got != tt.want {
				t.Errorf("Reduce() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterInPlace_String(t *testing.T) {
	type args struct {
		slice []string
		f     func(v string) bool
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "filter - empty",
			args: args{
				slice: []string{"one", "two", "three"},
				f: func(v string) bool {
					return false
				},
			},
			want: []string{},
		},
		{
			name: "filter - all",
			args: args{
				slice: []string{"one", "two", "three"},
				f: func(v string) bool {
					return true
				},
			},
			want: []string{"one", "two", "three"},
		},
		{
			name: "filter - some",
			args: args{
				slice: []string{"one", "two", "three"},
				f: func(v string) bool {
					return v != "two"
				},
			},
			want: []string{"one", "three"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FilterInPLace(tt.args.slice, tt.args.f); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FilterInPLace() = %v, want %v", got, tt.want)
			}
		})
	}
}
