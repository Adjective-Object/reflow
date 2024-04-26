package statemachine

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"testing"
)

type commandStepperTestCaseStep struct {
	inputByte byte
	command   Command
}

func (s commandStepperTestCaseStep) Compare(other commandStepperTestCaseStep) bool {
	if s.inputByte != other.inputByte {
		return false
	}

	if s.command.Type != other.command.Type {
		return false
	}

	if !bytes.Equal(s.command.CommandId, other.command.CommandId) {
		return false
	}

	if len(s.command.Params) != len(other.command.Params) {
		return false
	}

	for i, param := range s.command.Params {
		if !bytes.Equal(param, other.command.Params[i]) {
			return false
		}
	}

	return true
}

func printCommandStepChars(steps []commandStepperTestCaseStep, b *strings.Builder) {
	for _, step := range steps {
		asStr := strconv.Quote(string(step.inputByte))
		asStr = asStr[1 : len(asStr)-1]
		b.WriteString(asStr)

		for i := len(asStr); i < PAD_W; i++ {
			b.WriteString(" ")
		}
	}
	b.WriteString("\n")
}

func printCommandStepCommands(steps []commandStepperTestCaseStep, b *strings.Builder) {
	for i, step := range steps {
		if step.command.Type != 0 {
			asStr := fmt.Sprintf("%d:\t%+v\n  ", i, step.command)
			b.WriteString(asStr)
		} else {
			b.WriteString(fmt.Sprintf("%d:\n  ", i))
		}

	}
	b.WriteString("\n")
}

type commandStepperTestCase struct {
	steps []commandStepperTestCaseStep
}

func runCommandStepperTest(t *testing.T, testCase commandStepperTestCase) {
	stepper := CommandCollector{}
	realSteps := []commandStepperTestCaseStep{}

	input := make([]byte, len(testCase.steps))
	for i, step := range testCase.steps {
		input[i] = step.inputByte
	}
	inputText := string(input)

	if len(inputText) != len(testCase.steps) {
		t.Fatalf("mismatched input (%d) & expected output (%d) lengths", len(inputText), len(testCase.steps))
	}

	mismatched := false
	for i := 0; i < len(inputText); i++ {
		step := stepper.Next(
			inputText[i],
		)
		testStep := commandStepperTestCaseStep{
			inputByte: inputText[i],
			command:   step.Command,
		}
		if !testCase.steps[i].Compare(testStep) {
			mismatched = true
		}
		realSteps = append(realSteps, testStep)
	}

	if mismatched {
		err := strings.Builder{}
		err.WriteString("\nchars: ")
		printCommandStepChars(testCase.steps, &err)
		err.WriteString("       ")
		printCommandStepChars(realSteps, &err)

		err.WriteString("commands:\n  ")
		printCommandStepCommands(testCase.steps, &err)
		err.WriteString("real commands:\n  ")
		printCommandStepCommands(realSteps, &err)

		t.Error(err.String())
	}
}

func TestCollectCSICommand(t *testing.T) {
	runCommandStepperTest(t, commandStepperTestCase{
		steps: []commandStepperTestCaseStep{
			{
				inputByte: '\x1b',
			},
			{
				inputByte: '[',
			},
			{
				inputByte: '0',
			},
			{
				inputByte: 'm',
				command: Command{
					Type:      TypeCSICommand,
					CommandId: []byte("0m"),
				},
			},
		},
	})
}

func TestCollectOSCCommand(t *testing.T) {
	t.Run("terminated with BEL", func(t *testing.T) {
		runCommandStepperTest(t, commandStepperTestCase{
			steps: []commandStepperTestCaseStep{
				{
					inputByte: '\x1b',
				},
				{
					inputByte: ']',
				},
				{
					inputByte: '8',
				},
				{
					inputByte: ';',
				},
				{
					inputByte: 'f',
				},
				{
					inputByte: 'o',
				},
				{
					inputByte: 'o',
				},
				{
					inputByte: ':',
				},
				{
					inputByte: 'b',
				},
				{
					inputByte: 'a',
				},
				{
					inputByte: 'r',
				},
				{
					inputByte: '\x07',
					command: Command{
						Type:      TypeOSCCommand,
						CommandId: []byte("8"),
						Params:    [][]byte{[]byte("foo"), []byte("bar")},
					},
				},
			},
		})
	})

	t.Run("terminated with \\", func(t *testing.T) {
		runCommandStepperTest(t, commandStepperTestCase{
			steps: []commandStepperTestCaseStep{
				{
					inputByte: '\x1b',
				},
				{
					inputByte: ']',
				},
				{
					inputByte: '8',
				},
				{
					inputByte: ';',
				},
				{
					inputByte: 'f',
				},
				{
					inputByte: 'o',
				},
				{
					inputByte: 'o',
				},
				{
					inputByte: ':',
				},
				{
					inputByte: 'b',
				},
				{
					inputByte: 'a',
				},
				{
					inputByte: 'r',
				},
				{
					inputByte: '\\',
					command: Command{
						Type:      TypeOSCCommand,
						CommandId: []byte("8"),
						Params:    [][]byte{[]byte("foo"), []byte("bar")},
					},
				},
			},
		})
	})
}
