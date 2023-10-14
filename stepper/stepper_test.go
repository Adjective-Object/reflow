package stepper

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/muesli/reflow/ansi"
)

func TestKnownSequencesAreNotEscaping(t *testing.T) {
	t.Parallel()

	for i, seq := range KNOWN_SEQUENCES {
		for _, r := range seq.Sequence {
			if ansi.IsTerminator(r) {
				t.Errorf("sequence %d (%s) contains an escape character: %s",
					i,
					strconv.Quote(seq.Sequence),
					strconv.Quote(string(r)),
				)
			}
		}
	}
}

type stepperTestCaseStep struct {
	inputByte  byte
	afterState state
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
	runTest(
		t,
		stepperTestCase{
			steps: []stepperTestCaseStep{
				{' ', none, true},
				{'\x1b', gatheringEscapeSequence, false},
				{'[', cSICommand, false},
				{'4', cSICommand, false},
				{'m', none, true},
			},
		},
	)
}

func TestStepUnknownEarlyTermSequence(t *testing.T) {
	runTest(
		t,
		stepperTestCase{
			steps: []stepperTestCaseStep{
				{' ', none, true},
				{'\x1b', gatheringEscapeSequence, false},
				{'M', none, true},
				{':', none, true},
				{'3', none, true},
			},
		},
	)
}

func TestStepUnknownLongSequence(t *testing.T) {
	runTest(
		t,
		stepperTestCase{
			steps: []stepperTestCaseStep{
				{' ', none, true},
				{'\x1b', gatheringEscapeSequence, false},
				{'4', gatheringEscapeSequence, false},
				{'4', gatheringEscapeSequence, false},
				{'4', gatheringEscapeSequence, false},
				{'4', unknown, false},
				{'4', unknown, false},
				{'M', none, true},
				{' ', none, true},
			},
		},
	)
}

func TestStepLink(t *testing.T) {
	runTest(
		t,
		stepperTestCase{
			steps: []stepperTestCaseStep{
				{'h', none, true},
				{'i', none, true},
				{' ', none, true},
				{'\x1b', gatheringEscapeSequence, false},
				{']', oSCCommandId, false},
				{'8', oSCCommandId, false},
				{';', oSCParam, false},
				{';', oSCParam, false},
				{'h', oSCParam, false},
				{'t', oSCParam, false},
				{'t', oSCParam, false},
				{'p', oSCParam, false},
				{':', oSCParam, false},
				{'/', oSCParam, false},
				{'/', oSCParam, false},
				{'g', oSCParam, false},
				{'i', oSCParam, false},
				{'t', oSCParam, false},
				{'h', oSCParam, false},
				{'u', oSCParam, false},
				{'b', oSCParam, false},
				{'.', oSCParam, false},
				{'c', oSCParam, false},
				{'o', oSCParam, false},
				{'m', oSCParam, false},
				{'\x07', none, true},
				{'t', none, true},
				{'e', none, true},
				{'x', none, true},
				{'t', none, true},
				{'\x1b', gatheringEscapeSequence, false},
				{']', oSCCommandId, false},
				{'8', oSCCommandId, false},
				{';', oSCParam, false},
				{';', oSCParam, false},
				{'\x07', none, true},
			},
		},
	)
}
