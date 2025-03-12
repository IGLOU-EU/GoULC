package duration_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	json_helper "gitlab.com/iglou.eu/goulc/duration"
)

func TestDurationUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Duration
		wantErr bool
		err     string
	}{
		{
			name:    "valid string duration",
			input:   `{"duration": "1h30m"}`,
			want:    90 * time.Minute,
			wantErr: false,
		},
		{
			name:    "valid integer milliseconds",
			input:   `{"duration": 1000000000}`,
			want:    time.Millisecond * 1000,
			wantErr: false,
		},
		{
			name:    "valid float milliseconds",
			input:   `{"duration": 1000000000.5}`,
			want:    time.Millisecond * 1000,
			wantErr: false,
		},
		{
			name:    "valid negative duration",
			input:   `{"duration": "-2s"}`,
			want:    -2 * time.Second,
			wantErr: false,
		},
		{
			name:    "invalid duration string",
			input:   `{"duration": "1ho"}`,
			want:    0,
			wantErr: true,
			err:     "time: unknown unit",
		},
		{
			name:    "invalid type",
			input:   `{"duration": true}`,
			want:    0,
			wantErr: true,
			err:     json_helper.ErrDurationInvalidType.Error(),
		},
		{
			name:    "invalid json",
			input:   `{"duration": these aren't the JSON you're looking for}`,
			want:    0,
			wantErr: true,
			err:     "invalid character",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var obj struct {
				Duration json_helper.Duration `json:"duration"`
			}
			err := json.Unmarshal([]byte(tt.input), &obj)

			if (err != nil) != tt.wantErr {
				t.Errorf("An error occurred = %v, error was expected = %v", err, tt.wantErr)
				return
			}

			if err != nil && !strings.Contains(err.Error(), tt.err) {
				t.Errorf("Errors does not match = %v, want %v", err, tt.err)
				return
			}

			if err != nil && obj.Duration.ToTimeDuration() != tt.want {
				t.Errorf("= %v, want %v", obj.Duration.ToTimeDuration(), tt.want)
			}
		})
	}
}

func TestDurationMarshalJSON(t *testing.T) {
	tests := []struct {
		name  string
		input time.Duration
		want  string
	}{
		{
			name:  "regular duration",
			input: 90 * time.Minute,
			want:  `"1h30m0s"`,
		},
		{
			name:  "negative duration",
			input: -2 * time.Second,
			want:  `"-2s"`,
		},
		{
			name:  "zero duration",
			input: 0,
			want:  `"0s"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := json_helper.Duration{Duration: tt.input}
			got, err := json.Marshal(d)

			t.Log(tt.name)

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

func TestDurationToTimeDuration(t *testing.T) {
	v := 90 * time.Minute
	d := json_helper.Duration{Duration: v}

	got := d.ToTimeDuration()

	if got != v {
		t.Errorf("There is an issue during the conversion = %v, want %v", got, v)
	}

	if &d.Duration == &got {
		t.Errorf("Got value is supposed to be a copy of the original duration. Pointer to original value is %v, pointer to got value is %v", &d.Duration, &got)
	}
}
