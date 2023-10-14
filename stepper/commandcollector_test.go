package stepper

import (
	"strings"
	"testing"
)

type commandStepperTestCaseStep struct {
	inputByte byte
	command   Command
}
type commandStepperTestCase struct {
	steps []stepperTestCaseStep
}

func runCommandStepperTest(t *testing.T, testCase stepperTestCase) {
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
