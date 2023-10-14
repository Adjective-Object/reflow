package stepper

import "github.com/muesli/reflow/ansi"

type state int

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

	cSICommand

	// unrecognized state
	unknown
)

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
		if b == '\x07' {
			return none, true
		}
	default:
		if ansi.IsTerminator(rune(b)) {
			return none, true
		}
	}
	return s, false
}

type knownSequence struct {
	Sequence string
	state    state
}

var KNOWN_SEQUENCES = [2]knownSequence{
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

type StepperStep struct {
	prevState state
	nextState state
	isChange  bool
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
		if b == ansi.Marker {
			return s.changeState(gatheringEscapeSequence)
		}
	case gatheringEscapeSequence:
		// if this was a terminator, abort sequence recognition;
		// no sequences should contain a sequence-terminating character
		if ansi.IsTerminator(rune(b)) {
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
