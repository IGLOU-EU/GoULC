package jsonl

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"
)

type spaceLog struct {
	ID         int      `json:"id"`
	Character  string   `json:"character"`
	Quote      string   `json:"quote"`
	Tags       []string `json:"tags,omitempty"`
	Year       int      `json:"year,omitempty"`
	IsMajorTom bool     `json:"is_major_tom"`
	GroundCtrl string   `json:"ground_control,omitempty"`
}

func TestMarshal(t *testing.T) {
	tests := []struct {
		name    string
		input   []spaceLog
		want    []byte
		wantErr bool
	}{
		{
			name:    "nil slice returns nil",
			input:   nil,
			want:    nil,
			wantErr: false,
		},
		{
			name:    "empty slice returns nil",
			input:   []spaceLog{},
			want:    nil,
			wantErr: false,
		},
		{
			name: "single record HAL 9000",
			input: []spaceLog{
				{ID: 1, Character: "HAL 9000", Quote: "I'm sorry, Dave. I'm afraid I can't do that."},
			},
			want:    []byte(`{"id":1,"character":"HAL 9000","quote":"I'm sorry, Dave. I'm afraid I can't do that.","is_major_tom":false}` + "\n"),
			wantErr: false,
		},
		{
			name: "multiple mixed records",
			input: []spaceLog{
				{ID: 2, Character: "David Bowman", Quote: "My God, it's full of stars!", Year: 2001, IsMajorTom: false},
				{ID: 3, Character: "Major Tom", Quote: "Ground Control to Major Tom", Tags: []string{"space", "oddity"}, IsMajorTom: true, GroundCtrl: "Ground Control"},
				{ID: 4, Character: "Dave", Quote: "Open the pod bay doors, HAL.", IsMajorTom: false},
			},
			want: []byte(
				`{"id":2,"character":"David Bowman","quote":"My God, it's full of stars!","year":2001,"is_major_tom":false}` + "\n" +
					`{"id":3,"character":"Major Tom","quote":"Ground Control to Major Tom","tags":["space","oddity"],"is_major_tom":true,"ground_control":"Ground Control"}` + "\n" +
					`{"id":4,"character":"Dave","quote":"Open the pod bay doors, HAL.","is_major_tom":false}` + "\n",
			),
			wantErr: false,
		},
		{
			name: "record with special characters",
			input: []spaceLog{
				{ID: 5, Character: "Major Tom", Quote: "Planet Earth is blue, and there's nothing I can do.\n\nFor here am I sitting in a tin can.", IsMajorTom: true},
			},
			want:    []byte(`{"id":5,"character":"Major Tom","quote":"Planet Earth is blue, and there's nothing I can do.\n\nFor here am I sitting in a tin can.","is_major_tom":true}` + "\n"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Marshal(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Marshal() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !bytes.Equal(got, tt.want) {
				t.Errorf("Marshal() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUnmarshal(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    []spaceLog
		wantErr bool
		errIs   error
	}{
		{
			name:    "nil data returns nil",
			input:   nil,
			want:    nil,
			wantErr: false,
		},
		{
			name:    "empty data returns nil",
			input:   []byte{},
			want:    nil,
			wantErr: false,
		},
		{
			name:    "single line no trailing newline",
			input:   []byte(`{"id":1,"character":"HAL 9000","quote":"I'm sorry, Dave.","is_major_tom":false}`),
			want:    []spaceLog{{ID: 1, Character: "HAL 9000", Quote: "I'm sorry, Dave.", IsMajorTom: false}},
			wantErr: false,
		},
		{
			name:    "single line with trailing newline",
			input:   []byte(`{"id":1,"character":"HAL 9000","quote":"I'm sorry, Dave.","is_major_tom":false}` + "\n"),
			want:    []spaceLog{{ID: 1, Character: "HAL 9000", Quote: "I'm sorry, Dave.", IsMajorTom: false}},
			wantErr: false,
		},
		{
			name: "multiple lines mixed references",
			input: []byte(
				`{"id":2,"character":"David Bowman","quote":"My God, it's full of stars!","is_major_tom":false,"year":2001}` + "\n" +
					`{"id":3,"character":"Major Tom","quote":"Ground Control to Major Tom","is_major_tom":true,"tags":["space","oddity"],"ground_control":"Ground Control"}` + "\n",
			),
			want: []spaceLog{
				{ID: 2, Character: "David Bowman", Quote: "My God, it's full of stars!", Year: 2001, IsMajorTom: false},
				{ID: 3, Character: "Major Tom", Quote: "Ground Control to Major Tom", Tags: []string{"space", "oddity"}, IsMajorTom: true, GroundCtrl: "Ground Control"},
			},
			wantErr: false,
		},
		{
			name:    "blank line inside data returns error",
			input:   []byte(`{"id":1,"character":"HAL 9000","quote":"Daisy, Daisy...","is_major_tom":false}` + "\n\n" + `{"id":2,"character":"Major Tom","quote":"Tell my wife I love her very much.","is_major_tom":true}`),
			want:    nil,
			wantErr: true,
			errIs:   errors.New("jsonl: blank line inside JSONL data"),
		},
		{
			name:    "invalid JSON returns error",
			input:   []byte(`{"id":1,"character":"HAL 9000","quote":"I'm sorry, Dave.","is_major_tom":false`),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "empty line with trailing newline only",
			input:   []byte(`{"id":1,"character":"HAL 9000","quote":"I'm sorry, Dave.","is_major_tom":false}` + "\n" + "\n"),
			want:    nil,
			wantErr: true,
			errIs:   errors.New("jsonl: blank line inside JSONL data"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Unmarshal[spaceLog](tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.errIs != nil && err != nil {
				if err.Error() != tt.errIs.Error() {
					t.Errorf("Unmarshal() error = %v, want %v", err, tt.errIs)
				}
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Unmarshal() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestUnmarshalStream(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []spaceLog
		wantErr bool
		errIs   error
	}{
		{
			name:    "empty reader returns nil",
			input:   "",
			want:    nil,
			wantErr: false,
		},
		{
			name:    "single line no trailing newline",
			input:   `{"id":1,"character":"HAL 9000","quote":"I'm sorry, Dave.","is_major_tom":false}`,
			want:    []spaceLog{{ID: 1, Character: "HAL 9000", Quote: "I'm sorry, Dave.", IsMajorTom: false}},
			wantErr: false,
		},
		{
			name:    "single line with trailing newline",
			input:   `{"id":1,"character":"HAL 9000","quote":"I'm sorry, Dave.","is_major_tom":false}` + "\n",
			want:    []spaceLog{{ID: 1, Character: "HAL 9000", Quote: "I'm sorry, Dave.", IsMajorTom: false}},
			wantErr: false,
		},
		{
			name: "multiple lines mixed references",
			input: `{"id":2,"character":"David Bowman","quote":"My God, it's full of stars!","is_major_tom":false,"year":2001}` + "\n" +
				`{"id":3,"character":"Major Tom","quote":"Ground Control to Major Tom","is_major_tom":true,"tags":["space","oddity"],"ground_control":"Ground Control"}` + "\n",
			want: []spaceLog{
				{ID: 2, Character: "David Bowman", Quote: "My God, it's full of stars!", Year: 2001, IsMajorTom: false},
				{ID: 3, Character: "Major Tom", Quote: "Ground Control to Major Tom", Tags: []string{"space", "oddity"}, IsMajorTom: true, GroundCtrl: "Ground Control"},
			},
			wantErr: false,
		},
		{
			name:    "blank line inside data returns error",
			input:   `{"id":1,"character":"HAL 9000","quote":"Daisy, Daisy...","is_major_tom":false}` + "\n\n" + `{"id":2,"character":"Major Tom","quote":"Tell my wife I love her very much.","is_major_tom":true}`,
			want:    nil,
			wantErr: true,
			errIs:   errors.New("jsonl: blank line inside JSONL data"),
		},
		{
			name:    "invalid JSON returns error",
			input:   `{"id":1,"character":"HAL 9000","quote":"I'm sorry, Dave.","is_major_tom":false`,
			want:    nil,
			wantErr: true,
		},
		{
			name:    "empty line with trailing newline only",
			input:   `{"id":1,"character":"HAL 9000","quote":"I'm sorry, Dave.","is_major_tom":false}` + "\n" + "\n",
			want:    nil,
			wantErr: true,
			errIs:   errors.New("jsonl: blank line inside JSONL data"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := UnmarshalStream[spaceLog](strings.NewReader(tt.input))
			if (err != nil) != tt.wantErr {
				t.Fatalf("UnmarshalStream() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.errIs != nil && err != nil {
				if err.Error() != tt.errIs.Error() {
					t.Errorf("UnmarshalStream() error = %v, want %v", err, tt.errIs)
				}
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UnmarshalStream() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		input []spaceLog
	}{
		{
			name: "round trip HAL and Bowman",
			input: []spaceLog{
				{ID: 1, Character: "HAL 9000", Quote: "Daisy, Daisy, give me your answer do.", IsMajorTom: false},
				{ID: 2, Character: "David Bowman", Quote: "My God, it's full of stars!", Year: 2001, IsMajorTom: false},
			},
		},
		{
			name: "round trip Major Tom",
			input: []spaceLog{
				{ID: 3, Character: "Major Tom", Quote: "Here am I floating in my tin can.", Tags: []string{"space", "oddity", "bowie"}, IsMajorTom: true, GroundCtrl: "Ground Control"},
				{ID: 4, Character: "Frank Poole", Quote: "See you next Wednesday.", IsMajorTom: false},
			},
		},
		{
			name:  "round trip empty slice",
			input: []spaceLog{},
			// Marshal normalizes empty slice to nil
		},
		{
			name:  "round trip nil slice",
			input: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := Marshal(tt.input)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}

			decoded, err := Unmarshal[spaceLog](encoded)
			if err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}

			if tt.input != nil && len(tt.input) == 0 {
				// empty slice marshal→unmarshal normalizes to nil slice
				if len(decoded) != 0 {
					t.Errorf("round trip mismatch: got non-empty slice, want empty/nil")
				}
				return
			}
			if !reflect.DeepEqual(decoded, tt.input) {
				t.Errorf("round trip mismatch: got %+v, want %+v", decoded, tt.input)
			}
		})
	}
}

func TestRoundTripStream(t *testing.T) {
	tests := []struct {
		name  string
		input []spaceLog
	}{
		{
			name: "round trip HAL and Bowman",
			input: []spaceLog{
				{ID: 1, Character: "HAL 9000", Quote: "Daisy, Daisy, give me your answer do.", IsMajorTom: false},
				{ID: 2, Character: "David Bowman", Quote: "My God, it's full of stars!", Year: 2001, IsMajorTom: false},
			},
		},
		{
			name: "round trip Major Tom",
			input: []spaceLog{
				{ID: 3, Character: "Major Tom", Quote: "Here am I floating in my tin can.", Tags: []string{"space", "oddity", "bowie"}, IsMajorTom: true, GroundCtrl: "Ground Control"},
				{ID: 4, Character: "Frank Poole", Quote: "See you next Wednesday.", IsMajorTom: false},
			},
		},
		{
			name:  "round trip empty slice",
			input: []spaceLog{},
		},
		{
			name:  "round trip nil slice",
			input: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := Marshal(tt.input)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}

			decoded, err := UnmarshalStream[spaceLog](bytes.NewReader(encoded))
			if err != nil {
				t.Fatalf("UnmarshalStream() error = %v", err)
			}

			if tt.input != nil && len(tt.input) == 0 {
				if len(decoded) != 0 {
					t.Errorf("round trip mismatch: got non-empty slice, want empty/nil")
				}
				return
			}
			if !reflect.DeepEqual(decoded, tt.input) {
				t.Errorf("round trip mismatch: got %+v, want %+v", decoded, tt.input)
			}
		})
	}
}
