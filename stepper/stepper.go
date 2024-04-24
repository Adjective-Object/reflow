package stepper

// internal state of the stepper's state-machine
type state int

// Reference: https://gist.github.com/ConnerWill/d4b6c776b509add763e17f9f113fd25b
const (
	// not in any ansi escape sequence
	none state = iota

	// special state: we have seen \xb and are looking for
	// the following characters to recognize the escape sequence
	//
	// If we see a valid terminator, we will transition to
	// the corresponding knownEscapeSequence state.
	gatheringEscapeSequence

	// Collecting the ID of the OSC command
	// e.g. for xterm "link", this will be the "8" in
	// \x1b]8;;http://example.com\x07
	oSCCommandId
	oSCParam // a parameter to an OSC command

	// we are in a CSI command
	cSICommand

	// unrecognized state
	unknown
)

func (s state) String() string {
	switch s {
	case none:
		return "none"
	case gatheringEscapeSequence:
		return "gatheringEscapeSequence"
	case oSCCommandId:
		return "oSCCommandId"
	case oSCParam:
		return "oSCParam"
	case cSICommand:
		return "cSICommand"
	default:
		return "unknown"
	}
}

// True if this state is printing text
func (s state) HasPayload() bool {
	switch s {
	case oSCCommandId, oSCParam:
		return true
	default:
		return false
	}
}

// True if this state is printing text
func (s state) IsPrinting() bool {
	switch s {
	case none:
		return true
	case oSCCommandId, gatheringEscapeSequence, oSCParam, cSICommand:
		return false
	default:
		return false
	}
}

// Gets the next state for the given byte
func (s state) Step(b byte) (state, bool) {
	switch s {
	case oSCCommandId, oSCParam:
		if b == ';' {
			return oSCParam, true
		}
		// See https://en.wikipedia.org/wiki/ANSI_escape_code#OSC_(Operating_System_Command)_sequences
		// OSC sequences can be terminated by a BEL character or a ST character
		if b == '\x07' || b == '\\' {
			return none, true
		}
	default:
		if IsTerminatorByte(b) {
			return none, true
		}
	}
	return s, false
}

type knownSequence struct {
	Sequence string
	state    state
}

// See https://en.wikipedia.org/wiki/ANSI_escape_code#Fe_Escape_sequences
var KNOWN_SEQUENCES = [...]knownSequence{
	{"]", oSCCommandId},
	{"[", cSICommand},
}

// Stepper is used to step through ANSI escape sequences.
// and is used to determine the printable width of a string.
type Stepper struct {
	// state of the stepper
	state state

	// index into the current ansi identifier sequence
	ansiSeqIdx int

	// currently aggregating sequence
	ansiSeqPrefix [4]byte
}

// Represents the transition between two states
// as triggered by consuming a byte in a byte-sequence
type StepperStep struct {
	prevState state
	nextState state
	isChange  bool
}

func (s StepperStep) String() string {
	if s.isChange {
		return s.prevState.String() + " -> " + s.nextState.String()
	}
	return s.prevState.String() + " <no change>"

}

// If the character that triggered this transition should be printed
// or not
func (s *StepperStep) IsPrintingStep() bool {
	return s.nextState.IsPrinting() && s.prevState.IsPrinting()
}

// If this step is a transition between states
func (s *StepperStep) IsChange() bool {
	return s.isChange
}

func (s *Stepper) changeState(next state) StepperStep {
	step := StepperStep{
		prevState: s.state,
		nextState: next,
		isChange:  true,
	}
	s.state = next
	s.ansiSeqIdx = 0
	return step
}

func (s *Stepper) Next(b byte) StepperStep {
	switch s.state {
	case none:
		// if we are in normal text and see the ansi marker, start
		// trying to gather the escape sequence
		if b == Marker {
			return s.changeState(gatheringEscapeSequence)
		}
	case gatheringEscapeSequence:
		// if this was a terminator, abort sequence recognition;
		// no sequences should contain a sequence-terminating character
		if IsTerminatorByte(b) {
			return s.changeState(none)
		}

		// gather the sequence into the stepper
		s.ansiSeqPrefix[s.ansiSeqIdx] = b
		s.ansiSeqIdx++
		if s.ansiSeqIdx >= len(s.ansiSeqPrefix) {
			// overstepped max len of sequence - we failed recognition;
			// go to "unknown"
			s.state = unknown
			s.ansiSeqIdx = 0
			return StepperStep{s.state, s.state, false}
		} else {
			// otherwise, we are still gathering the sequence;
			// check each known sequence to see if we have a match
			for _, seq := range KNOWN_SEQUENCES {
				if len(seq.Sequence) == s.ansiSeqIdx && string(s.ansiSeqPrefix[:s.ansiSeqIdx]) == seq.Sequence {
					return s.changeState(seq.state)
				}
			}
		}
	default:
		// once we have a state, let it handle the byte
		nextState, isTransition := s.state.Step(b)
		if isTransition {
			return s.changeState(nextState)
		}
	}
	return StepperStep{s.state, s.state, false}
}
