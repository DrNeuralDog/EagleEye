package animation

import "time"

// DefaultConfig returns LeoEye-inspired defaults.
func DefaultConfig() Config {
	return Config{
		InstructionDuration: 2 * time.Second,
		CenterDuration: Range{
			Min: time.Second,
			Max: time.Second,
		},
		MoveDuration: Range{
			Min: 300 * time.Millisecond,
			Max: 400 * time.Millisecond,
		},
		HoldDuration: Range{
			Min: 1500 * time.Millisecond,
			Max: 2 * time.Second,
		},
		ReturnDuration: Range{
			Min: 300 * time.Millisecond,
			Max: 400 * time.Millisecond,
		},
		PauseDuration: Range{
			Min: 500 * time.Millisecond,
			Max: 800 * time.Millisecond,
		},
		BlinkClosedDuration: Range{
			Min: 150 * time.Millisecond,
			Max: 200 * time.Millisecond,
		},
		BlinkOpenDuration: Range{
			Min: 150 * time.Millisecond,
			Max: 200 * time.Millisecond,
		},
		BlinkInterval: Range{
			Min: 3 * time.Second,
			Max: 8 * time.Second,
		},
		DoubleBlinkChance: 0.12,
		DoubleBlinkGap: Range{
			Min: 50 * time.Millisecond,
			Max: 100 * time.Millisecond,
		},
		CombinedSwitchAfter: 15 * time.Second,
	}
}
