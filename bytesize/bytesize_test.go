package bytesize_test

import (
	"testing"

	"gitlab.com/iglou.eu/goulc/bytesize"
)

func TestByteSize_ArrayLiteral(t *testing.T) {
	if len(bytesize.ByteSymbolIEC) != len(bytesize.ByteValueIEC) {
		t.Fatalf(
			"ByteSymbolIEC and ByteValueIEC must have the same length\nByteSymbol (%v) %v\nByteValue (%v) %v",
			len(bytesize.ByteSymbolIEC),
			bytesize.ByteSymbolIEC,
			len(bytesize.ByteValueIEC),
			bytesize.ByteValueIEC,
		)
	}
}

func Test_Parse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wInt    int64
		wFloat  float64
		wString string
		wErr    string
	}{
		{
			name:    "empty string",
			input:   "",
			wInt:    0,
			wFloat:  0,
			wString: "",
			wErr:    bytesize.ErrEmptyString,
		},
		{
			name:    "implicit byte zero value",
			input:   "0",
			wInt:    0,
			wFloat:  0,
			wString: "0B",
		},
		{
			name:    "implicit byte value",
			input:   "1042",
			wInt:    1042,
			wFloat:  1042,
			wString: "1.02KiB",
		},
		{
			name:    "implicit negative byte value",
			input:   "-1042",
			wInt:    -1042,
			wFloat:  -1042,
			wString: "-1.02KiB",
		},
		{
			name:    "implicit floating byte value",
			input:   "42.42",
			wInt:    42,
			wFloat:  42.42,
			wString: "42B",
		},
		{
			name:    "gibi value",
			input:   "1042GiB",
			wInt:    1042 * bytesize.Gibi,
			wFloat:  float64(1042 * bytesize.Gibi),
			wString: "1.02TiB",
		},
		{
			name:    "kibi negative value",
			input:   "-1042KiB",
			wInt:    -1042 * bytesize.Kibi,
			wFloat:  float64(-1042 * bytesize.Kibi),
			wString: "-1.02MiB",
		},
		{
			name:    "mebi floating value",
			input:   "42.42MiB",
			wInt:    44480593,
			wFloat:  float64(44480593.92),
			wString: "42.42MiB",
		},
		{
			name:    "short symbol value",
			input:   "42M",
			wInt:    42 * bytesize.Mebi,
			wFloat:  float64(42 * bytesize.Mebi),
			wString: "42MiB",
		},
		{
			name:  "symbol without value",
			input: "PiB",
			wErr:  bytesize.ErrNoValue,
		},
		{
			name:  "invalid numeric value",
			input: "a Byte",
			wErr:  `strconv.ParseFloat: parsing "a Byte": invalid syntax`,
		},
		{
			name:  "invalid unit",
			input: `1Xor`,
			wErr:  bytesize.ErrInvalidIEC,
		},
		{
			name:  "too big value",
			input: "10000P",
			wErr:  bytesize.ErrIntegerOverflow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gInt, gFloat, gString, err := bytesize.Parse(tt.input)

			if (err != nil) != (tt.wErr != "") ||
				(err != nil && err.Error() != tt.wErr) {
				t.Errorf("Error does not match = %v, expected = %v", err, tt.wErr)
				return
			}

			if gInt != tt.wInt || gFloat != tt.wFloat || gString != tt.wString {
				t.Errorf("Result does not match\n Got: Int %d, Float %f, String %s\nWant: Int %d, Float %f, String %s", gInt, gFloat, gString, tt.wInt, tt.wFloat, tt.wString)
			}
		})
	}
}

func Test_ToString(t *testing.T) {
	tests := []struct {
		name  string
		input float64
		want  string
	}{
		{
			name:  "zero value",
			input: 0,
			want:  "0B",
		},
		{
			name:  "negative Byte value",
			input: -42,
			want:  "-42B",
		},
		{
			name:  "negative gibi value",
			input: float64(-42 * bytesize.Gibi),
			want:  "-42GiB",
		},
		{
			name:  "floating value",
			input: 42.42,
			want:  "42B",
		},
		{
			name:  "exbi value",
			input: float64(4200 * bytesize.Pebi),
			want:  "4200PiB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bytesize.ToString(tt.input)

			if got != tt.want {
				t.Errorf("Result does not match = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_New(t *testing.T) {
	_, err := bytesize.New("hoho!")

	if err == nil {
		t.Errorf("Error should not be nil")
	}

	s, err := bytesize.New("42M")

	if err != nil {
		t.Error("Error should be nil", err)
		return
	}

	if s.String() != "42MiB" {
		t.Errorf("Unexpected value = %s, want %s", s.String(), "42MiB")
	}

	s = bytesize.NewInt(42)

	if s.String() != "42B" {
		t.Errorf("Unexpected value = %s, want %s", s.String(), "42B")
	}

	if s.Bytes() != 42 {
		t.Errorf("Unexpected value = %d, want %d", s.Bytes(), 42)
	}

	if s.Exact() != 42.0 {
		t.Errorf("Unexpected value = %f, want %f", s.Exact(), 42.0)
	}
}

func Test_Add(t *testing.T) {
	tests := []struct {
		name   string
		base   string
		inSTR  string
		inINT  int64
		want   string
		errSTR string
		errINT string
	}{
		{
			name:  "zero value",
			base:  "42MiB",
			inSTR: "0",
			inINT: 0,
			want:  "42MiB",
		},
		{
			name:  "add byte",
			base:  "42MiB",
			inSTR: "100KiB",
			inINT: 100 * bytesize.Kibi,
			want:  "42.1MiB",
		},
		{
			name:  "add negative byte",
			base:  "42MiB",
			inSTR: "-100KiB",
			inINT: -100 * bytesize.Kibi,
			want:  "41.9MiB",
		},
		{
			name:  "add floating byte",
			base:  "42MiB",
			inSTR: "100.42KiB",
			inINT: 100 * bytesize.Kibi,
			want:  "42.1MiB",
		},
		{
			name:  "add negative floating byte",
			base:  "42MiB",
			inSTR: "-100.42KiB",
			inINT: -100 * bytesize.Kibi,
			want:  "41.9MiB",
		},
		{
			name:   "add too big value",
			base:   "8000PiB",
			inSTR:  "4200PiB",
			inINT:  4200 * bytesize.Pebi,
			errSTR: bytesize.ErrIntegerOverflow,
			errINT: bytesize.ErrIntegerOverflow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// String tests
			main, _ := bytesize.New(tt.base)

			err := main.Add(tt.inSTR)
			if (err != nil) != (tt.errSTR != "") ||
				(err != nil && err.Error() != tt.errSTR) {
				t.Errorf("STR Add error does not match = %v, expected = %v", err, tt.errSTR)
				return
			}

			if (tt.errSTR == "") && main.String() != tt.want {
				t.Errorf("STR Add result does not match = %v, expected = %v", main.String(), tt.want)
			}

			// Int tests
			main, _ = bytesize.New(tt.base)

			err = main.AddInt(tt.inINT)
			if (err != nil) != (tt.errINT != "") ||
				(err != nil && err.Error() != tt.errINT) {
				t.Errorf("INT Add error does not match = %v, expected = %v", err, tt.errINT)
				return
			}

			if (tt.errINT == "") && main.String() != tt.want {
				t.Errorf("INT Add result does not match = %v, expected = %v", main.String(), tt.want)
			}
		})
	}
}
