package stepper

type CommandType int

const (
	TypeOSCCommand CommandType = iota
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
	stepper         Stepper
	buildingCommand Command
	currentPayload  []byte
}

type CollectorStep struct {
	Command Command
	StepperStep
}

func (collector *CommandCollector) Next(b byte) CollectorStep {
	step := CollectorStep{
		StepperStep: collector.stepper.Next(b),
	}

	if step.isChange {
		// build the right param of the command
		switch step.prevState {
		case oSCCommandId:
			collector.buildingCommand.Type = TypeOSCCommand
			collector.buildingCommand.CommandId = string(collector.currentPayload)
			collector.currentPayload = nil
		case cSICommand:
			collector.buildingCommand.Type = TypeCSICommand
			collector.buildingCommand.CommandId = string(collector.currentPayload)
			collector.currentPayload = nil
		case oSCParam:
			collector.buildingCommand.Params = append(collector.buildingCommand.Params, string(collector.currentPayload))
			collector.currentPayload = nil
		}

		// if we terminated a command, return it and clear the stored command
		if step.nextState == none {
			if collector.buildingCommand.Type != 0 {
				step.Command = collector.buildingCommand
				collector.buildingCommand = Command{}
			}
		}
	} else if step.nextState.HasPayload() {
		// aggregate the payload
		collector.currentPayload = append(collector.currentPayload, b)
	}

	return step
}
