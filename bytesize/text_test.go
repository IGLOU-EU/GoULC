package bytesize_test

import (
	"encoding/json"
	"errors"
	"testing"

	"gitlab.com/iglou.eu/goulc/bytesize"
)

func TestByteSize_MarshalText(t *testing.T) {
	tests := []struct {
		name string
		give bytesize.Size
		want string
	}{
		{
			name: "zero value",
			give: bytesize.Size{},
			want: "0B",
		},
		{
			name: "exact kibibyte",
			give: bytesize.NewInt(bytesize.Kibi),
			want: "1KiB",
		},
		{
			name: "negative value",
			give: bytesize.NewInt(-42),
			want: "-42B",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.give.MarshalText()
			if err != nil {
				t.Errorf("An error occurred = %v", err)
				return
			}

			if string(got) != tt.want {
				t.Errorf("Result does not match = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestByteSize_UnmarshalText(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   int64
		wExact float64
		wErr   error
	}{
		{
			name:   "regular value",
			input:  "42MiB",
			want:   42 * bytesize.Mebi,
			wExact: float64(42 * bytesize.Mebi),
		},
		{
			name:   "floating value",
			input:  "1.9",
			want:   1,
			wExact: 1.9,
		},
		{
			name:  "empty value",
			input: "",
			wErr:  bytesize.ErrEmptyString,
		},
		{
			name:  "invalid unit",
			input: "1Xor",
			wErr:  bytesize.ErrInvalidIEC,
		},
		{
			name:  "infinite value",
			input: "inf",
			wErr:  bytesize.ErrIntegerOverflow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got bytesize.Size
			err := got.UnmarshalText([]byte(tt.input))

			if !errors.Is(err, tt.wErr) {
				t.Errorf("Error does not match = %v, want %v", err, tt.wErr)
				return
			}

			if err == nil &&
				(got.Bytes() != tt.want || got.Exact() != tt.wExact) {
				t.Errorf(
					"Result does not match = %v (exact %v), want %v (exact %v)",
					got.Bytes(), got.Exact(), tt.want, tt.wExact,
				)
			}
		})
	}
}

func TestByteSize_TextMarshaling_MapKey(t *testing.T) {
	// Text marshaling is what lets a Size act as a JSON map key.
	give := map[bytesize.Size]int{
		bytesize.NewInt(bytesize.Kibi): 1,
	}

	data, err := json.Marshal(give)
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}

	if want := `{"1KiB":1}`; string(data) != want {
		t.Errorf("Result does not match = %s, want %s", data, want)
	}

	got := make(map[bytesize.Size]int)
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if got[bytesize.NewInt(bytesize.Kibi)] != 1 {
		t.Errorf("Round-trip lost the entry, got %v", got)
	}
}
