package ascii

import "testing"

var tests = []struct {
	name         string
	str          string
	fIs          bool
	fIsPrintable bool
	fIsExtended  bool
	fHasNil      bool
}{
	{
		name:         "empty string",
		str:          "",
		fIs:          true,
		fIsPrintable: true,
		fIsExtended:  true,
		fHasNil:      false,
	},
	{
		name:         "regular ASCII string",
		str:          "I'll be back!",
		fIs:          true,
		fIsPrintable: true,
		fIsExtended:  true,
		fHasNil:      false,
	},
	{
		name:         "emoji reaction",
		str:          "Use the Force 🚀 of emoji",
		fIs:          false,
		fIsPrintable: false,
		fIsExtended:  false,
		fHasNil:      false,
	},
	{
		name:         "null terminator",
		str:          "Winter is coming\x00And so are the nil byte",
		fIs:          true,
		fIsPrintable: false,
		fIsExtended:  true,
		fHasNil:      true,
	},
	{
		name:         "extended ASCII",
		str:          "Pokémon & Pikachu ÿ Catch'em all!",
		fIs:          false,
		fIsPrintable: false,
		fIsExtended:  true,
		fHasNil:      false,
	},
	{
		name:         "control characters",
		str:          "To infinity\nand beyond!",
		fIs:          true,
		fIsPrintable: false,
		fIsExtended:  true,
		fHasNil:      false,
	},
	{
		name:         "mixed content",
		str:          "Matrix\x00🕶️ÿReloaded",
		fIs:          false,
		fIsPrintable: false,
		fIsExtended:  false,
		fHasNil:      true,
	},
	{
		name:         "multiple nil bytes",
		str:          "Hasta\x00la\x00vista\x00babyte",
		fIs:          true,
		fIsPrintable: false,
		fIsExtended:  true,
		fHasNil:      true,
	},
	{
		name:         "all control chars",
		str:          "\n\r\t\b",
		fIs:          true,
		fIsPrintable: false,
		fIsExtended:  true,
		fHasNil:      false,
	},
	{
		name:         "sequential extended ASCII",
		str:          "Pokémon évolution: Pikachu » Raichu",
		fIs:          false,
		fIsPrintable: false,
		fIsExtended:  true,
		fHasNil:      false,
	},
}

func TestHasNil(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasNil(tt.str); got != tt.fHasNil {
				t.Errorf("HasNil(%q) = %v, want %v", tt.str, got, tt.fHasNil)
			}
		})
	}
}

func TestIs(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Is(tt.str); got != tt.fIs {
				t.Errorf("Is(%q) = %v, want %v", tt.str, got, tt.fIs)
			}
		})
	}
}

func TestIsPrintable(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsPrintable(tt.str); got != tt.fIsPrintable {
				t.Errorf("IsPrintable(%q) = %v, want %v", tt.str, got, tt.fIsPrintable)
			}
		})
	}
}

func TestIsExtended(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsExtended(tt.str); got != tt.fIsExtended {
				t.Errorf("IsExtended(%q) = %v, want %v", tt.str, got, tt.fIsExtended)
			}
		})
	}
}
