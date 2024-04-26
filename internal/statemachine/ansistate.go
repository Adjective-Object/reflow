package statemachine

import "bytes"

// represents the state of an ansi sequence at any given point in time
//
// This is used to determine what, if any, an ansi Reset() should look like
// for the given sequence.
type AnsiState struct {
	colorCmd             Command
	lastXtermLinkCommand Command
	collector            CommandCollector
}

func (ansiState *AnsiState) Next(b byte) CollectorStep {
	step := ansiState.collector.Next(b)
	if step.Command.Type == TypeCSICommand {
		if bytes.Equal(step.Command.CommandId, []byte{'0', 'm'}) {
			// Reset color sequence
			ansiState.colorCmd = Command{}
		} else if bytes.HasSuffix(step.Command.CommandId, []byte{'m'}) {
			// Some non-reset color sequence -- we may need to reset
			// at the end of the sequence
			ansiState.colorCmd = step.Command
		}
	} else if step.Command.Type == TypeOSCCommand && bytes.Equal(step.Command.CommandId, []byte{'8'}) {
		if isResetLinkParams(step.Command.Params) {
			// this is a reset xterm link command
			ansiState.lastXtermLinkCommand = Command{}
		} else {
			ansiState.lastXtermLinkCommand = step.Command
		}
	}

	return step
}

func isResetLinkParams(params [][]byte) bool {
	return len(params) < 2 || len(params[1]) == 0
}

// Gets an ansi sequence that reproduces the stored state of the AnsiState
func (ansiState *AnsiState) RestoreSequence() []byte {
	cap := 0
	if ansiState.colorCmd.Type != TypeNone {
		cap += len(ansiState.colorCmd.CommandId) + 2
	}
	if ansiState.lastXtermLinkCommand.Type != TypeNone {
		cap += len(ansiState.lastXtermLinkCommand.CommandId) + 2
		for _, param := range ansiState.lastXtermLinkCommand.Params {
			cap += len(param) + 1
		}
	}
	seq := make([]byte, 0, cap)
	if ansiState.colorCmd.Type != TypeNone {
		seq = append(seq, []byte("\x1b[")...)
		seq = append(seq, ansiState.colorCmd.CommandId...)
	}
	if ansiState.lastXtermLinkCommand.Type != TypeNone {
		seq = append(seq, []byte("\x1b]")...)
		seq = append(seq, ansiState.lastXtermLinkCommand.CommandId...)
		for _, param := range ansiState.lastXtermLinkCommand.Params {
			seq = append(seq, ';')
			seq = append(seq, param...)
		}
		seq = append(seq, '\x1b')
		seq = append(seq, '\\')
	}

	return seq
}

const colorResetSeq = "\x1b[0m"
const xtermResetSeq = "\x1b]8;;\x1b\\"

// Gets an ansi sequence that can reset a stream to a neutral state
// against the stored state of the AnsiState
func (ansiState *AnsiState) ResetSequence() []byte {
	cap := 0
	if ansiState.colorCmd.Type != TypeNone {
		cap += len(colorResetSeq)
	}
	if ansiState.lastXtermLinkCommand.Type != TypeNone {
		cap += len(xtermResetSeq)
	}

	seq := make([]byte, 0, cap)
	if ansiState.colorCmd.Type != TypeNone {
		seq = append(seq, colorResetSeq...)
	}
	if ansiState.lastXtermLinkCommand.Type != TypeNone {
		seq = append(seq, xtermResetSeq...)
	}

	return seq
}

// Clears the internal state of the ansistate
func (ansiState *AnsiState) ClearState() {
	if ansiState.colorCmd.Type == TypeCSICommand {
		// Reset color sequence
		_ = ansiState.collector.Next('m')
	}
	if ansiState.lastXtermLinkCommand.Type == TypeOSCCommand {
		// Reset xterm link
		_ = ansiState.collector.Next('\x07')
	}
}

// Returns true if the AnsiState has any state that would require a reset
func (ansiState *AnsiState) IsDirty() bool {
	return ansiState.colorCmd.Type != TypeNone || ansiState.lastXtermLinkCommand.Type != TypeNone
}
