package statemachine

// internal State of the stepper's State-machine
type State byte

// Reference: https://gist.github.com/ConnerWill/d4b6c776b509add763e17f9f113fd25b
const (
	// not in any ansi escape sequence
	nonAnsi State = iota << 0

	// unrecognized state
	unknown

	// special state: we have seen \x1b and are looking for
	// the following characters to recognize the escape sequence
	//
	// If we see a valid terminator, we will transition to either
	// oscCommandID or csiCommand
	gatheringEscapeSequence

	// payload states go below here:

	// Collecting the ID of the OSC command
	// e.g. for xterm "link", this will be the "8" in
	// \x1b]8;;http://example.com\x07
	oscCommandID
	oscParameter // a parameter to an OSC command

	// we are in a CSI command
	csiCommand

	maxState
)

func (s State) String() string {
	switch s {
	case nonAnsi:
		return "none"
	case gatheringEscapeSequence:
		return "gatheringEscapeSequence"
	case oscCommandID:
		return "oSCCommandId"
	case oscParameter:
		return "oSCParam"
	case csiCommand:
		return "cSICommand"
	default:
		return "unknown"
	}
}

// True if this state has a "payload" -- text which should be gathered
// in order to understand the command
//
// See: CommandCollector
func (s State) HasPayload() bool {
	return s >= oscCommandID
}

// True if this state is printing text
func (s State) IsPrinting() bool {
	return s == nonAnsi
}

// Gets the next state for the given byte
func (s State) Step(b byte) State {
	switch s {
	case nonAnsi:
		// if we are in normal text and see the ansi marker, start
		// trying to gather the escape sequence
		if b == Marker {
			return gatheringEscapeSequence
		}
	case gatheringEscapeSequence:
		// if this was a terminator, abort sequence recognition;
		// no sequences should contain a sequence-terminating character
		if IsTerminatorByte(b) {
			return nonAnsi
		}

		switch b {
		case ']':
			return oscCommandID
		case '[':
			return csiCommand
		default:
			// we are in an unknown sequence
			return unknown
		}
	case oscCommandID, oscParameter:
		if b == ';' {
			return oscParameter
		}
		// See https://en.wikipedia.org/wiki/ANSI_escape_code#OSC_(Operating_System_Command)_sequences
		// OSC sequences can be terminated by a BEL character or a ST character
		if b == '\x07' || b == '\\' {
			return nonAnsi
		}
	default:
		if IsTerminatorByte(b) {
			return nonAnsi
		}
	}
	return s
}

// StateMachine is used to step through ANSI escape sequences.
// and is used to determine the printable width of a string.
type StateMachine struct {
	// state of the stepper
	state State
}

// Represents the transition between two states
// as triggered by consuming a byte in a byte-sequence
type StateTransition byte

// If the character that triggered this transition should be printed
// or not
func (s *StateTransition) IsPrinting() bool {
	return State(*s) == nonAnsi
}

func (s *StateMachine) changeState(next State) StateTransition {
	step := StateTransition(s.state | next)
	s.state = next
	return step
}

// Advances the state machine by one byte
func (s *StateMachine) Next(b byte) StateTransition {
	return s.changeState(s.state.Step(b))
}
