package bytesize_test

import (
	"encoding/json"
	"errors"
	"strconv"
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
			name:   "scientific notation number",
			input:  `1E5`,
			want:   100000,
			wExact: 100000,
		},
		{
			name:  "too big number value",
			input: `1e300`,
			wErr:  bytesize.ErrIntegerOverflow,
		},
		{
			// 1e400 overflows float64 itself, ParseFloat rejects it with
			// ErrRange, exactly as it would for the "1e400" string form.
			name:  "number out of float64 range",
			input: `1e400`,
			wErr:  strconv.ErrRange,
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

func TestByteSize_UnmarshalJSON_MalformedJSON(t *testing.T) {
	// Malformed JSON never reaches the value switch, the decoder must
	// surface the syntax error straight away.
	var s bytesize.Size
	if err := s.UnmarshalJSON([]byte(`{`)); err == nil {
		t.Error("expected a decode error for malformed JSON, got nil")
	}
}

func TestByteSize_UnmarshalJSON_NumberStringParity(t *testing.T) {
	// A JSON number and the same literal quoted as a JSON string must decode
	// to a byte-identical Size: the number branch shares the string branch's
	// Parse path, so the fraction and the canonical form never diverge.
	literals := []string{"1.9", "44480593.92", "-44040192", "1E5", "100"}

	for _, lit := range literals {
		t.Run(lit, func(t *testing.T) {
			var fromNumber, fromString bytesize.Size

			if err := json.Unmarshal([]byte(lit), &fromNumber); err != nil {
				t.Fatalf("number unmarshal error = %v", err)
			}

			if err := json.Unmarshal([]byte(`"`+lit+`"`), &fromString); err != nil {
				t.Fatalf("string unmarshal error = %v", err)
			}

			if fromNumber != fromString {
				t.Errorf(
					"number/string mismatch for %q: bytes %d/%d exact %v/%v repr %q/%q",
					lit,
					fromNumber.Bytes(), fromString.Bytes(),
					fromNumber.Exact(), fromString.Exact(),
					fromNumber.String(), fromString.String(),
				)
			}
		})
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
