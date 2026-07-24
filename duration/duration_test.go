package duration_test

import (
	"encoding/json"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"gitlab.com/iglou.eu/goulc/duration"
)

func TestDuration_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name        string
		give        string
		want        time.Duration
		wantErr     bool
		wantErrIs   error
		wantErrText string
	}{
		{
			name: "string duration",
			give: `{"duration": "1h30m"}`,
			want: 90 * time.Minute,
		},
		{
			name: "negative string duration",
			give: `{"duration": "-2s"}`,
			want: -2 * time.Second,
		},
		{
			name: "integer nanoseconds",
			give: `{"duration": 1000000000}`,
			want: time.Second,
		},
		{
			name: "negative integer nanoseconds",
			give: `{"duration": -1000000000}`,
			want: -time.Second,
		},
		{
			name: "integer above float64 precision",
			give: `{"duration": 9007199254740993}`,
			want: time.Duration(9007199254740993),
		},
		{
			name: "maximum int64 nanoseconds",
			give: `{"duration": 9223372036854775807}`,
			want: time.Duration(math.MaxInt64),
		},
		{
			name:      "number overflowing int64",
			give:      `{"duration": 9223372036854775808}`,
			wantErr:   true,
			wantErrIs: duration.ErrBadDuration,
		},
		{
			name:      "huge scientific notation number",
			give:      `{"duration": 1e300}`,
			wantErr:   true,
			wantErrIs: duration.ErrBadDuration,
		},
		{
			name:      "fractional number",
			give:      `{"duration": 1000000000.5}`,
			wantErr:   true,
			wantErrIs: duration.ErrBadDuration,
		},
		{
			name:        "invalid duration string",
			give:        `{"duration": "1ho"}`,
			wantErr:     true,
			wantErrIs:   duration.ErrBadDuration,
			wantErrText: "time: unknown unit",
		},
		{
			name:      "boolean value",
			give:      `{"duration": true}`,
			wantErr:   true,
			wantErrIs: duration.ErrDurationInvalidType,
		},
		{
			name:      "array value",
			give:      `{"duration": [1]}`,
			wantErr:   true,
			wantErrIs: duration.ErrDurationInvalidType,
		},
		{
			name:        "invalid json",
			give:        `{"duration": these aren't the JSON you're looking for}`,
			wantErr:     true,
			wantErrText: "invalid character",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var obj struct {
				Duration duration.Duration `json:"duration"`
			}

			err := json.Unmarshal([]byte(tt.give), &obj)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
				t.Fatalf("Unmarshal() error = %v, want errors.Is %v",
					err, tt.wantErrIs)
			}
			if tt.wantErrText != "" &&
				!strings.Contains(err.Error(), tt.wantErrText) {
				t.Fatalf("Unmarshal() error = %q, want it to contain %q",
					err, tt.wantErrText)
			}
			if err == nil && obj.Duration.Duration != tt.want {
				t.Errorf("Duration = %v, want %v", obj.Duration.Duration, tt.want)
			}
		})
	}
}

func TestDuration_UnmarshalJSON_Null(t *testing.T) {
	obj := struct {
		Duration duration.Duration `json:"duration"`
	}{Duration: duration.Duration{Duration: 5 * time.Second}}

	if err := json.Unmarshal([]byte(`{"duration": null}`), &obj); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got := obj.Duration.Duration; got != 5*time.Second {
		t.Errorf("null changed the value to %v, want it left untouched", got)
	}
}

func TestDuration_MarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		give time.Duration
		want string
	}{
		{
			name: "regular duration",
			give: 90 * time.Minute,
			want: `"1h30m0s"`,
		},
		{
			name: "negative duration",
			give: -2 * time.Second,
			want: `"-2s"`,
		},
		{
			name: "zero duration",
			give: 0,
			want: `"0s"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(duration.Duration{Duration: tt.give})
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("Marshal() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestDuration_JSONRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		give time.Duration
	}{
		{name: "zero", give: 0},
		{name: "composite duration", give: time.Hour + 30*time.Minute},
		{name: "negative duration", give: -2 * time.Second},
		{name: "sub-second precision", give: 1234567890 * time.Nanosecond},
		{name: "maximum duration", give: time.Duration(math.MaxInt64)},
		{name: "minimum duration", give: time.Duration(math.MinInt64)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := json.Marshal(duration.Duration{Duration: tt.give})
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}

			var got duration.Duration
			if err := json.Unmarshal(b, &got); err != nil {
				t.Fatalf("Unmarshal(%s) error = %v", b, err)
			}
			if got.Duration != tt.give {
				t.Errorf("round trip = %v, want %v", got.Duration, tt.give)
			}
		})
	}
}

func TestDuration_ToTimeDuration(t *testing.T) {
	want := 90 * time.Minute

	if got := (duration.Duration{Duration: want}).ToTimeDuration(); got != want {
		t.Errorf("ToTimeDuration() = %v, want %v", got, want)
	}
}
