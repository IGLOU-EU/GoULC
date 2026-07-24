package ascii

import "testing"

var tests = []struct {
	name            string
	str             string
	wantIs          bool
	wantIsPrintable bool
	wantIsExtended  bool
	wantHasNil      bool
}{
	{
		name:            "empty string",
		str:             "",
		wantIs:          true,
		wantIsPrintable: true,
		wantIsExtended:  true,
		wantHasNil:      false,
	},
	{
		name:            "regular ASCII string",
		str:             "I'll be back!",
		wantIs:          true,
		wantIsPrintable: true,
		wantIsExtended:  true,
		wantHasNil:      false,
	},
	{
		name:            "highest printable character",
		str:             "~",
		wantIs:          true,
		wantIsPrintable: true,
		wantIsExtended:  true,
		wantHasNil:      false,
	},
	{
		name:            "DEL control character",
		str:             "Game over\x7fman",
		wantIs:          true,
		wantIsPrintable: false,
		wantIsExtended:  true,
		wantHasNil:      false,
	},
	{
		name:            "emoji reaction",
		str:             "Use the Force 🚀 of emoji",
		wantIs:          false,
		wantIsPrintable: false,
		wantIsExtended:  false,
		wantHasNil:      false,
	},
	{
		name:            "null terminator",
		str:             "Winter is coming\x00And so are the nil byte",
		wantIs:          true,
		wantIsPrintable: false,
		wantIsExtended:  true,
		wantHasNil:      true,
	},
	{
		name:            "extended ASCII",
		str:             "Pokémon & Pikachu ÿ Catch'em all!",
		wantIs:          false,
		wantIsPrintable: false,
		wantIsExtended:  true,
		wantHasNil:      false,
	},
	{
		name:            "control characters",
		str:             "To infinity\nand beyond!",
		wantIs:          true,
		wantIsPrintable: false,
		wantIsExtended:  true,
		wantHasNil:      false,
	},
	{
		name:            "mixed content",
		str:             "Matrix\x00🕶️ÿReloaded",
		wantIs:          false,
		wantIsPrintable: false,
		wantIsExtended:  false,
		wantHasNil:      true,
	},
	{
		name:            "multiple nil bytes",
		str:             "Hasta\x00la\x00vista\x00babyte",
		wantIs:          true,
		wantIsPrintable: false,
		wantIsExtended:  true,
		wantHasNil:      true,
	},
	{
		name:            "all control chars",
		str:             "\n\r\t\b",
		wantIs:          true,
		wantIsPrintable: false,
		wantIsExtended:  true,
		wantHasNil:      false,
	},
	{
		name:            "sequential extended ASCII",
		str:             "Pokémon évolution: Pikachu » Raichu",
		wantIs:          false,
		wantIsPrintable: false,
		wantIsExtended:  true,
		wantHasNil:      false,
	},
}

func TestHasNil(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasNil(tt.str); got != tt.wantHasNil {
				t.Errorf("HasNil(%q) = %v, want %v", tt.str, got, tt.wantHasNil)
			}
		})
	}
}

func TestIs(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Is(tt.str); got != tt.wantIs {
				t.Errorf("Is(%q) = %v, want %v", tt.str, got, tt.wantIs)
			}
		})
	}
}

func TestIsPrintable(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsPrintable(tt.str); got != tt.wantIsPrintable {
				t.Errorf("IsPrintable(%q) = %v, want %v", tt.str, got, tt.wantIsPrintable)
			}
		})
	}
}

func TestIsExtended(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsExtended(tt.str); got != tt.wantIsExtended {
				t.Errorf("IsExtended(%q) = %v, want %v", tt.str, got, tt.wantIsExtended)
			}
		})
	}
}
