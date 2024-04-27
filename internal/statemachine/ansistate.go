package statemachine

import (
	"bytes"
)

// represents the state of an ansi sequence at any given point in time
//
// This is used to determine what, if any, an ansi Reset() should look like
// for the given sequence.
type AnsiState struct {
	colorCmd         Command
	xtermLinkCommand Command
	collector        CommandCollector
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
			ansiState.xtermLinkCommand = Command{}
		} else {
			ansiState.xtermLinkCommand = step.Command
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
	if ansiState.xtermLinkCommand.Type != TypeNone {
		cap += len(ansiState.xtermLinkCommand.CommandId) + 2
		for _, param := range ansiState.xtermLinkCommand.Params {
			cap += len(param) + 1
		}
	}
	buf := bytes.NewBuffer(make([]byte, 0, cap))
	ansiState.WriteRestoreSequence(buf)
	return buf.Bytes()
}

// Gets an ansi sequence that can reset a stream to a neutral state
// against the stored state of the AnsiState
func (ansiState *AnsiState) WriteRestoreSequence(out *bytes.Buffer) {
	if ansiState.colorCmd.Type != TypeNone {
		// bytes.Buffer's write operations don't err, so we can ignore
		// all these errors. They're only in the signature because they
		// implement the io.*Writer interfaces
		out.WriteString("\x1b[")
		out.Write(ansiState.colorCmd.CommandId)
	}
	if ansiState.xtermLinkCommand.Type != TypeNone {
		out.WriteString("\x1b]")
		out.Write(ansiState.xtermLinkCommand.CommandId)
		for _, param := range ansiState.xtermLinkCommand.Params {
			out.WriteRune(';')
			out.Write(param)
		}
		out.WriteString("\x1b\\")
	}
}

// Gets an ansi sequence that can reset a stream to a neutral state
// against the stored state of the AnsiState
func (ansiState *AnsiState) ResetSequence() []byte {
	cap := 0
	if ansiState.colorCmd.Type != TypeNone {
		cap += len(colorResetSeq)
	}
	if ansiState.xtermLinkCommand.Type != TypeNone {
		cap += len(xtermResetSeq1) + len(xtermResetSeq2)
		if len(ansiState.xtermLinkCommand.Params) > 0 {
			cap += len(ansiState.xtermLinkCommand.Params[0])
		}
	}

	buf := bytes.NewBuffer(make([]byte, 0, cap))
	ansiState.WriteResetSequence(buf)
	return buf.Bytes()
}

const colorResetSeq = "\x1b[0m"
const xtermResetSeq1 = "\x1b]8;"
const xtermResetSeq2 = ";\x1b\\"

// Gets an ansi sequence that can reset a stream to a neutral state
// against the stored state of the AnsiState
func (ansiState *AnsiState) WriteResetSequence(out *bytes.Buffer) {
	if ansiState.colorCmd.Type != TypeNone {
		out.WriteString(colorResetSeq)
	}
	if ansiState.xtermLinkCommand.Type != TypeNone {
		out.WriteString(xtermResetSeq1)
		// write any link parameters
		if len(ansiState.xtermLinkCommand.Params) > 0 &&
			len(ansiState.xtermLinkCommand.Params[0]) > 0 {
			out.Write(ansiState.xtermLinkCommand.Params[0])
		}
		out.WriteString(xtermResetSeq2)
	}
}

// Clears the internal state of the ansistate
func (ansiState *AnsiState) ClearState() {
	ansiState.colorCmd = Command{}
	ansiState.xtermLinkCommand = Command{}
}

// Returns true if the AnsiState has any state that would require a reset
func (ansiState *AnsiState) IsDirty() bool {
	return ansiState.colorCmd.Type != TypeNone || ansiState.xtermLinkCommand.Type != TypeNone
}
