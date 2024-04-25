package statemachine

type CommandType int

const (
	TypeNone CommandType = iota
	TypeOSCCommand
	TypeCSICommand
)

// CommandCollector is used to collect ANSI commands
// with their parmaeters from the state transitions
// produced by the stepper.
type Command struct {
	Type      CommandType
	CommandId string
	Params    []string
}

type CommandCollector struct {
	stepper         StateMachine
	buildingCommand Command
	currentPayload  []byte
}

type CollectorStep struct {
	Command Command
	StateTransition
}

func (collector *CommandCollector) Next(b byte) CollectorStep {
	step := CollectorStep{
		StateTransition: collector.stepper.Next(b),
	}

	if step.isChange {
		// build the right param of the command
		switch step.prevState {
		case oscCommandID:
			collector.buildingCommand.Type = TypeOSCCommand
			collector.buildingCommand.CommandId = string(collector.currentPayload)
			collector.currentPayload = nil
		case csiCommand:
			collector.currentPayload = append(collector.currentPayload, b)
			collector.buildingCommand.Type = TypeCSICommand
			collector.buildingCommand.CommandId = string(collector.currentPayload)
			collector.currentPayload = nil
		case oscParameter:
			collector.buildingCommand.Params = append(collector.buildingCommand.Params, string(collector.currentPayload))
			collector.currentPayload = nil
		}

		// if we terminated a command, return it and clear the stored command
		if step.nextState == nonAnsi {
			if collector.buildingCommand.Type != 0 {
				step.Command = collector.buildingCommand
				collector.buildingCommand = Command{}
			}
		}
	} else if step.nextState == oscParameter && b == ':' {
		// if we're in oscParameter and we hit a colon, we're about to start a new parameter
		collector.buildingCommand.Params = append(collector.buildingCommand.Params, string(collector.currentPayload))
		collector.currentPayload = nil
	} else {
		if step.nextState.HasPayload() {
			// aggregate the payload
			collector.currentPayload = append(collector.currentPayload, b)
		}
	}

	return step
}
