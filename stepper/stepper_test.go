package stepper

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

func TestKnownSequencesAreNotEscaping(t *testing.T) {
	t.Parallel()

	for i, seq := range KNOWN_SEQUENCES {
		for _, b := range []byte(seq.Sequence) {
			if IsTerminatorByte(b) {
				t.Errorf("sequence %d (%s) contains an escape character: %s",
					i,
					strconv.Quote(seq.Sequence),
					strconv.Quote(string(rune(b))),
				)
			}
		}
	}
}

type stepperTestCaseStep struct {
	inputByte  byte
	afterState State
	printing   bool
}
type stepperTestCase struct {
	steps []stepperTestCaseStep
}

func printStepChars(steps []stepperTestCaseStep, b *strings.Builder) {
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

const PAD_W = 4

func printPrintable(steps []stepperTestCaseStep, b *strings.Builder) {
	for _, step := range steps {
		var c string
		if !step.printing {
			c = "x"
		}
		b.WriteString(c)

		for i := len(c); i < PAD_W; i++ {
			b.WriteString(" ")
		}
	}
	b.WriteString("\n")
}

func printState(steps []stepperTestCaseStep, b *strings.Builder) {
	for _, step := range steps {
		stateStr := fmt.Sprintf("%d", step.afterState)
		b.WriteString(stateStr)
		for i := len(stateStr); i < PAD_W; i++ {
			b.WriteString(" ")
		}
	}
	b.WriteString("\n")
}

func runStepperTest(t *testing.T, testCase stepperTestCase) {
	stepper := Stepper{}
	realSteps := []stepperTestCaseStep{}

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
		testStep := stepperTestCaseStep{
			inputByte:  inputText[i],
			afterState: step.nextState,
			printing:   step.nextState.IsPrinting(),
		}
		if testStep != testCase.steps[i] {
			mismatched = true
		}
		realSteps = append(realSteps, testStep)
	}

	if mismatched {
		err := strings.Builder{}
		err.WriteString("\nchars: ")
		printStepChars(testCase.steps, &err)
		err.WriteString("       ")
		printStepChars(realSteps, &err)

		err.WriteString("print: ")
		printPrintable(testCase.steps, &err)
		err.WriteString("       ")
		printPrintable(realSteps, &err)

		err.WriteString("state: ")
		printState(testCase.steps, &err)
		err.WriteString("       ")
		printState(realSteps, &err)

		t.Error(err.String())
	}
}

func TestStepCSISequence(t *testing.T) {
	runStepperTest(
		t,
		stepperTestCase{
			steps: []stepperTestCaseStep{
				{' ', nonAnsi, true},
				{'\x1b', gatheringEscapeSequence, false},
				{'[', csiCommand, false},
				{'4', csiCommand, false},
				{'m', nonAnsi, true},
			},
		},
	)
}

func TestStepUnknownEarlyTermSequence(t *testing.T) {
	runStepperTest(
		t,
		stepperTestCase{
			steps: []stepperTestCaseStep{
				{' ', nonAnsi, true},
				{'\x1b', gatheringEscapeSequence, false},
				{'M', nonAnsi, true},
				{':', nonAnsi, true},
				{'3', nonAnsi, true},
			},
		},
	)
}

func TestStepUnknownLongSequence(t *testing.T) {
	runStepperTest(
		t,
		stepperTestCase{
			steps: []stepperTestCaseStep{
				{' ', nonAnsi, true},
				{'\x1b', gatheringEscapeSequence, false},
				{'4', gatheringEscapeSequence, false},
				{'4', gatheringEscapeSequence, false},
				{'4', gatheringEscapeSequence, false},
				{'4', unknown, false},
				{'4', unknown, false},
				{'M', nonAnsi, true},
				{' ', nonAnsi, true},
			},
		},
	)
}

func TestStepLink(t *testing.T) {
	runStepperTest(
		t,
		stepperTestCase{
			steps: []stepperTestCaseStep{
				{'h', nonAnsi, true},
				{'i', nonAnsi, true},
				{' ', nonAnsi, true},
				{'\x1b', gatheringEscapeSequence, false},
				{']', oscCommandID, false},
				{'8', oscCommandID, false},
				{';', oscParameter, false},
				{';', oscParameter, false},
				{'h', oscParameter, false},
				{'t', oscParameter, false},
				{'t', oscParameter, false},
				{'p', oscParameter, false},
				{':', oscParameter, false},
				{'/', oscParameter, false},
				{'/', oscParameter, false},
				{'g', oscParameter, false},
				{'i', oscParameter, false},
				{'t', oscParameter, false},
				{'h', oscParameter, false},
				{'u', oscParameter, false},
				{'b', oscParameter, false},
				{'.', oscParameter, false},
				{'c', oscParameter, false},
				{'o', oscParameter, false},
				{'m', oscParameter, false},
				{'\x07', nonAnsi, true},
				{'t', nonAnsi, true},
				{'e', nonAnsi, true},
				{'x', nonAnsi, true},
				{'t', nonAnsi, true},
				{'\x1b', gatheringEscapeSequence, false},
				{']', oscCommandID, false},
				{'8', oscCommandID, false},
				{';', oscParameter, false},
				{';', oscParameter, false},
				{'\x07', nonAnsi, true},
			},
		},
	)
}
