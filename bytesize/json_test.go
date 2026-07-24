package bytesize_test

import (
	"encoding/json"
	"errors"
	"testing"

	"gitlab.com/iglou.eu/goulc/bytesize"
)

func TestByteSize_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int64
		wErr  error
	}{
		// Number cases
		{
			name:  "integer value",
			input: `44040192`,
			want:  44040192,
		},
		{
			name:  "floating value",
			input: `44480593.92`,
			want:  44480593,
		},
		{
			name:  "negative value",
			input: `-44040192`,
			want:  -44040192,
		},

		// String cases
		{
			name:  "regular str value",
			input: `"42MiB"`,
			want:  42 * bytesize.Mebi,
		},
		{
			name:  "invalid unit type",
			input: `"1Xor le chérif de l'espace"`,
			wErr:  bytesize.ErrInvalidIEC,
		},

		// Other error cases
		{
			name:  "invalid json type",
			input: `["Minsc", "Boo"]`,
			wErr:  bytesize.ErrJSONInvalidType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got bytesize.Size
			err := json.Unmarshal([]byte(tt.input), &got)

			if !errors.Is(err, tt.wErr) {
				t.Errorf("Error does not match = %v, want %v", err, tt.wErr)
				return
			}

			if err == nil && got.Bytes() != tt.want {
				t.Errorf("Result does not match = %v, want %v", got.Bytes(), tt.want)
			}
		})
	}
}

func TestByteSize_MarshalJSON(t *testing.T) {
	tests := []struct {
		name  string
		input bytesize.Size
		want  string
	}{
		{
			name:  "zero",
			input: bytesize.NewInt(0),
			want:  `"0B"`,
		},
		{
			name:  "exact petabyte",
			input: bytesize.NewInt(bytesize.Pebi),
			want:  `"1PiB"`,
		},
		{
			name:  "negative value",
			input: bytesize.NewInt(-42),
			want:  `"-42B"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.input)
			if err != nil {
				t.Errorf("An error occurred = %v", err)
				return
			}

			if string(got) != tt.want {
				t.Errorf("Result does not match = %v, want %v", string(got), tt.want)
			}
		})
	}
}
