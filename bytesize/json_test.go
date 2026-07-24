package bytesize_test

import (
	"encoding/json"
	"errors"
	"testing"

	"gitlab.com/iglou.eu/goulc/bytesize"
)

func TestByteSize_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   int64
		wExact float64
		wErr   error
	}{
		// Number cases
		{
			name:   "integer value",
			input:  `44040192`,
			want:   44040192,
			wExact: 44040192,
		},
		{
			name:   "floating value",
			input:  `44480593.92`,
			want:   44480593,
			wExact: 44480593.92,
		},
		{
			name:   "negative value",
			input:  `-44040192`,
			want:   -44040192,
			wExact: -44040192,
		},
		{
			name:  "too big number value",
			input: `1e300`,
			wErr:  bytesize.ErrIntegerOverflow,
		},

		// String cases
		{
			name:   "regular str value",
			input:  `"42MiB"`,
			want:   42 * bytesize.Mebi,
			wExact: float64(42 * bytesize.Mebi),
		},
		{
			name:   "floating str value",
			input:  `"1.9"`,
			want:   1,
			wExact: 1.9,
		},
		{
			name:  "invalid unit type",
			input: `"1Xor le chérif de l'espace"`,
			wErr:  bytesize.ErrInvalidIEC,
		},
		{
			name:  "infinite str value",
			input: `"inf"`,
			wErr:  bytesize.ErrIntegerOverflow,
		},
		{
			name:  "negative infinite str value with unit",
			input: `"-infKiB"`,
			wErr:  bytesize.ErrIntegerOverflow,
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

func TestByteSize_UnmarshalJSON_InvalidNumber(t *testing.T) {
	// 1e400 is valid JSON syntax but does not fit a float64, so the
	// decoding inside UnmarshalJSON must surface the json error.
	var got bytesize.Size
	err := json.Unmarshal([]byte(`1e400`), &got)

	var typeErr *json.UnmarshalTypeError
	if !errors.As(err, &typeErr) {
		t.Errorf("Error is not a json.UnmarshalTypeError = %v", err)
	}
}

func TestByteSize_OmitZero(t *testing.T) {
	// IsZero drives the ",omitzero" tag option, a zero Size field must
	// disappear from the JSON output.
	give := struct {
		Used bytesize.Size `json:"used,omitzero"`
		Free bytesize.Size `json:"free,omitzero"`
	}{Used: bytesize.NewInt(bytesize.Kibi)}

	data, err := json.Marshal(give)
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}

	if want := `{"used":"1KiB"}`; string(data) != want {
		t.Errorf("Result does not match = %s, want %s", data, want)
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
