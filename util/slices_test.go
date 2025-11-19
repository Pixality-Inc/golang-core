package util

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

var errTestError = errors.New("error at 2")

func TestMap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   []int
		f       func(x int) (string, error)
		want    []string
		wantErr bool
	}{
		{
			name:  "empty slice",
			input: []int{},
			f: func(x int) (string, error) {
				return "", nil
			},
			want:    nil,
			wantErr: false,
		},
		{
			name:  "single element",
			input: []int{5},
			f: func(x int) (string, error) {
				return "5", nil
			},
			want:    []string{"5"},
			wantErr: false,
		},
		{
			name:  "multiple elements",
			input: []int{1, 2, 3, 4, 5},
			f: func(x int) (string, error) {
				return string(rune('a' + x - 1)), nil
			},
			want:    []string{"a", "b", "c", "d", "e"},
			wantErr: false,
		},
		{
			name:  "error in mapping",
			input: []int{1, 2, 3},
			f: func(x int) (string, error) {
				if x == 2 {
					return "", errTestError
				}

				return "", nil
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			got, err := Map(testcase.input, testcase.f)
			if testcase.wantErr {
				require.Error(t, err)
				require.Nil(t, got)
			} else {
				require.NoError(t, err)
				require.Equal(t, testcase.want, got)
			}
		})
	}
}

func TestMapSimple(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []int
		f     func(x int) string
		want  []string
	}{
		{
			name:  "empty slice",
			input: []int{},
			f: func(x int) string {
				return ""
			},
			want: nil,
		},
		{
			name:  "single element",
			input: []int{5},
			f: func(x int) string {
				return "5"
			},
			want: []string{"5"},
		},
		{
			name:  "multiple elements",
			input: []int{1, 2, 3, 4, 5},
			f: func(x int) string {
				return string(rune('a' + x - 1))
			},
			want: []string{"a", "b", "c", "d", "e"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := MapSimple(tt.input, tt.f)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestSliceUnique(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []int
		want  []int
	}{
		{
			name:  "empty slice",
			input: []int{},
			want:  []int{},
		},
		{
			name:  "single element",
			input: []int{1},
			want:  []int{1},
		},
		{
			name:  "no duplicates",
			input: []int{1, 2, 3, 4, 5},
			want:  []int{1, 2, 3, 4, 5},
		},
		{
			name:  "all duplicates",
			input: []int{1, 1, 1, 1},
			want:  []int{1},
		},
		{
			name:  "some duplicates",
			input: []int{1, 2, 2, 3, 3, 3, 4},
			want:  []int{1, 2, 3, 4},
		},
		{
			name:  "duplicates not adjacent",
			input: []int{1, 2, 3, 1, 2, 3},
			want:  []int{1, 2, 3},
		},
	}

	for _, testcase := range tests {
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			got := SliceUnique(testcase.input)
			// Since order is not guaranteed, we need to check if all elements are present
			require.ElementsMatch(t, testcase.want, got)
		})
	}
}

func TestSliceSort(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []int
		comp  func(int, int) bool
		want  []int
	}{
		{
			name:  "empty slice",
			input: []int{},
			comp: func(a, b int) bool {
				return a < b
			},
			want: []int{},
		},
		{
			name:  "single element",
			input: []int{5},
			comp: func(a, b int) bool {
				return a < b
			},
			want: []int{5},
		},
		{
			name:  "ascending order",
			input: []int{5, 2, 8, 1, 9},
			comp: func(a, b int) bool {
				return a < b
			},
			want: []int{1, 2, 5, 8, 9},
		},
		{
			name:  "descending order",
			input: []int{5, 2, 8, 1, 9},
			comp: func(a, b int) bool {
				return a > b
			},
			want: []int{9, 8, 5, 2, 1},
		},
		{
			name:  "already sorted",
			input: []int{1, 2, 3, 4, 5},
			comp: func(a, b int) bool {
				return a < b
			},
			want: []int{1, 2, 3, 4, 5},
		},
	}

	for _, testcase := range tests {
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			// Make a copy to avoid modifying the original
			got := make([]int, len(testcase.input))
			copy(got, testcase.input)

			SliceSort(got, testcase.comp)
			require.Equal(t, testcase.want, got)
		})
	}
}
