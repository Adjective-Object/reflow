package statemachine

type CommandType int

const (
	TypeNone CommandType = iota
	TypeOSCCommand
	TypeCSICommand
)

// a Command represents a single ANSI command
type Command struct {
	Type CommandType
	// The command ID.
	// For OSC commands, this is the command number.
	// For CSI commands, this is the full body of the command
	CommandId []byte
	// For OSC commands, this is the parameters to the command,
	// separated by semicolons.
	//
	// For CSI commands, this will always be `nil`
	Params [][]byte
}

// CommandCollector is used to collect ANSI commands
// with their parameters from the state transitions
// produced by the stepper.
type CommandCollector struct {
	stepper         StateMachine
	buildingCommand Command
	currentPayload  []byte
}

// Wrapper for the state transition produced by the stepper
// that also includes the command (if any) produced by the
// step.
type CollectorStep struct {
	Command Command
	StateTransition
}

func (collector *CommandCollector) Next(b byte) CollectorStep {
	prev := collector.stepper.state
	step := CollectorStep{
		StateTransition: collector.stepper.Next(b),
	}
	next := collector.stepper.state

	if prev != next {
		// build the right param of the command
		switch prev {
		case oscCommandID:
			collector.buildingCommand.Type = TypeOSCCommand
			collector.buildingCommand.CommandId = collector.currentPayload
			collector.currentPayload = nil
		case csiCommand:
			collector.currentPayload = append(collector.currentPayload, b)
			collector.buildingCommand.Type = TypeCSICommand
			collector.buildingCommand.CommandId = collector.currentPayload
			collector.currentPayload = nil
		case oscParameter:
			collector.buildingCommand.Params = append(collector.buildingCommand.Params, collector.currentPayload)
			collector.currentPayload = nil
		}

		// if we terminated a command, return it and clear the stored command
		if next == nonAnsi {
			if collector.buildingCommand.Type != 0 {
				step.Command = collector.buildingCommand
				collector.buildingCommand = Command{}
			}
		}
	} else if next == oscParameter && b == ':' {
		// if we're in oscParameter and we hit a colon, we're about to start a new parameter
		collector.buildingCommand.Params = append(collector.buildingCommand.Params, collector.currentPayload)
		collector.currentPayload = nil
	} else {
		if next.HasPayload() {
			// aggregate the payload
			collector.currentPayload = append(collector.currentPayload, b)
		}
	}

	return step
}
