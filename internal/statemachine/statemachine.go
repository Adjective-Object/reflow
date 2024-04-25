package statemachine

// internal State of the stepper's State-machine
type State int

// Reference: https://gist.github.com/ConnerWill/d4b6c776b509add763e17f9f113fd25b
const (
	// not in any ansi escape sequence
	nonAnsi State = iota

	// special state: we have seen \x1b and are looking for
	// the following characters to recognize the escape sequence
	//
	// If we see a valid terminator, we will transition to
	// the corresponding knownEscapeSequence state.
	gatheringEscapeSequence

	// Collecting the ID of the OSC command
	// e.g. for xterm "link", this will be the "8" in
	// \x1b]8;;http://example.com\x07
	oscCommandID
	oscParameter // a parameter to an OSC command

	// we are in a CSI command
	csiCommand

	// unrecognized state
	unknown
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

// True if this state is printing text
func (s State) HasPayload() bool {
	switch s {
	case oscCommandID, oscParameter, csiCommand:
		return true
	default:
		return false
	}
}

// True if this state is printing text
func (s State) IsPrinting() bool {
	switch s {
	case nonAnsi:
		return true
	case oscCommandID, gatheringEscapeSequence, oscParameter, csiCommand:
		return false
	default:
		return false
	}
}

// Gets the next state for the given byte
func (s State) Step(b byte) (State, bool) {
	switch s {
	case oscCommandID, oscParameter:
		if b == ';' {
			return oscParameter, true
		}
		// See https://en.wikipedia.org/wiki/ANSI_escape_code#OSC_(Operating_System_Command)_sequences
		// OSC sequences can be terminated by a BEL character or a ST character
		if b == '\x07' || b == '\\' {
			return nonAnsi, true
		}
	default:
		if IsTerminatorByte(b) {
			return nonAnsi, true
		}
	}
	return s, false
}

// StateMachine is used to step through ANSI escape sequences.
// and is used to determine the printable width of a string.
type StateMachine struct {
	// state of the stepper
	state State
}

// Represents the transition between two states
// as triggered by consuming a byte in a byte-sequence
type StateTransition struct {
	prevState State
	nextState State
	isChange  bool
}

func (s StateTransition) String() string {
	if s.isChange {
		return s.prevState.String() + " -> " + s.nextState.String()
	}
	return s.prevState.String() + " <no change>"
}

// If the character that triggered this transition should be printed
// or not
func (s *StateTransition) IsPrintingStep() bool {
	return s.nextState.IsPrinting() && s.prevState.IsPrinting()
}

// If the character that triggered this transition should be printed
// or not
func (s *StateTransition) PreviousState() State {
	return s.prevState
}

// If the character that triggered this transition should be printed
// or not
func (s *StateTransition) NextState() State {
	return s.nextState
}

// If this step is a transition between states
func (s *StateTransition) IsChange() bool {
	return s.isChange
}

func (s *StateMachine) changeState(next State) StateTransition {
	step := StateTransition{
		prevState: s.state,
		nextState: next,
		isChange:  true,
	}
	s.state = next
	return step
}

func (s *StateMachine) Next(b byte) StateTransition {
	switch s.state {
	case nonAnsi:
		// if we are in normal text and see the ansi marker, start
		// trying to gather the escape sequence
		if b == Marker {
			return s.changeState(gatheringEscapeSequence)
		}
	case gatheringEscapeSequence:
		// if this was a terminator, abort sequence recognition;
		// no sequences should contain a sequence-terminating character
		if IsTerminatorByte(b) {
			return s.changeState(nonAnsi)
		}

		switch b {
		case ']':
			return s.changeState(oscCommandID)
		case '[':
			return s.changeState(csiCommand)
		default:
			// we are in an unknown sequence
			s.state = unknown
			return StateTransition{s.state, s.state, false}
		}
	default:
		// once we have a state, let it handle the byte
		nextState, isTransition := s.state.Step(b)
		if isTransition {
			return s.changeState(nextState)
		}
	}
	return StateTransition{s.state, s.state, false}
}