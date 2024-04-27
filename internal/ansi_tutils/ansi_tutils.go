package ansi_tutils

import (
	"bytes"
	"io"
	"reflect"
	"strconv"
	"testing"
)

type TestCase struct {
	Input    string
	Expected string
	Params   interface{}
}

type WriterWithBuffer interface {
	io.Writer
	Bytes() []byte
	String() string
}

type TestFunc = func(t testing.TB, writer io.Writer, params interface{}) WriterWithBuffer

func assertOutputs(t *testing.T, w WriterWithBuffer, output string) {
	gotString := w.String()
	if gotString != output {
		t.Errorf("expected:\n\n`%s`\n\nActual Output:\n\n`%s`",
			strconv.Quote(output),
			strconv.Quote(gotString))
	}

	gotBytes := w.Bytes()
	if string(gotBytes) != output {
		t.Errorf("expected:\n\n`%s`\n\nActual Output:\n\n`%s`",
			strconv.Quote(output),
			strconv.Quote(string(gotBytes)))
	}
}

type RuneWriter interface {
	WriteRune(r rune) (int, error)
}

func runIndividualTest(
	t *testing.T,
	fwdWriter io.Writer,
	testFunc TestFunc,
	tc TestCase,
) {
	// Check .Write()
	writer := testFunc(t, fwdWriter, tc.Params)
	if _, err := writer.Write([]byte(tc.Input)); err != nil {
		t.Error(err)
	}
	assertOutputs(t, writer, tc.Expected)

	// also check for WriteString()
	if _, is := writer.(io.StringWriter); is {
		t.Run(".WriteString", func(t *testing.T) {
			writer := testFunc(t, fwdWriter, tc.Params)
			sw := writer.(io.StringWriter)
			if _, err := sw.WriteString(tc.Input); err != nil {
				t.Fatal(err)
			}
			assertOutputs(t, writer, tc.Expected)
		})
	}

	// also check for WriteByte()
	if _, is := writer.(io.ByteWriter); is {
		t.Run(".WriteByte", func(t *testing.T) {
			writer := testFunc(t, fwdWriter, tc.Params)
			sw := writer.(io.ByteWriter)
			for i := 0; i < len(tc.Input); i++ {
				if err := sw.WriteByte(tc.Input[i]); err != nil {
					t.Fatal(err)
				}
			}
			assertOutputs(t, writer, tc.Expected)
		})
	}

	// also check for WriteRune()
	if _, is := writer.(RuneWriter); is {
		t.Run(".WriteRunes", func(t *testing.T) {
			writer := testFunc(t, fwdWriter, tc.Params)
			sw := writer.(RuneWriter)
			for _, r := range tc.Input {
				if _, err := sw.WriteRune(r); err != nil {
					t.Fatal(err)
				}
			}
			assertOutputs(t, writer, tc.Expected)
		})
	}
}

func RunTests(
	t *testing.T,
	testCases []TestCase,
	writerFactory TestFunc,
) {
	for i, tc := range testCases {
		tc := tc
		i := i

		t.Run("test case "+strconv.Itoa(i), func(t *testing.T) {
			t.Parallel()
			t.Run("fwd", func(t *testing.T) {
				t.Parallel()
				forwardBuffer := &bytes.Buffer{}
				writer := writerFactory(t, forwardBuffer, tc.Params)
				if _, err := writer.Write([]byte(tc.Input)); err != nil {
					t.Fatal(err)
				}
				got := forwardBuffer.String()
				if got != tc.Expected {
					t.Errorf("for input:\n%s\n\nexpected:\n`%s`\n\nActual Output:\n`%s`",
						strconv.Quote(tc.Input),
						strconv.Quote(tc.Expected),
						strconv.Quote(got))
				}
			})

			t.Run("buf", func(t *testing.T) {
				t.Parallel()
				writer := writerFactory(t, nil, tc.Params)
				if _, err := writer.Write([]byte(tc.Input)); err != nil {
					t.Fatal(err)
				}
				assertOutputs(t, writer, tc.Expected)
			})
		})
	}
}

// Big reflection-based hack to fuzz that the forward and buffer outputs are the same
// for all testcases.
func RunFuzzEq(
	t *testing.F,
	testCases []TestCase,
	testFunc TestFunc,
) {

	typ := reflect.TypeOf(testCases[0].Params)
	typArgs := []reflect.Type{
		reflect.TypeOf((*testing.T)(nil)),
		reflect.TypeOf("")}
	for i := 0; i < typ.NumField(); i++ {
		if isUnsupported(typ.Field(i).Type.Kind()) {
			typArgs = append(typArgs, reflect.TypeOf(1))
		} else {
			typArgs = append(typArgs, typ.Field(i).Type)
		}
	}

	fc := 0
	funcs := map[int]interface{}{}

	// convert structs to fuzz args
	for _, tc := range testCases {
		val := reflect.ValueOf(tc.Params)
		var params []interface{}
		params = append(params, tc.Input)
		for i := 0; i < typ.NumField(); i++ {
			if isUnsupported(val.Field(i).Kind()) {
				funcs[fc] = val.Field(i).Interface()
				params = append(params, fc)
				fc++
			} else {
				params = append(params, val.Field(i).Interface())
			}
		}

		t.Add(params...)
	}

	f := reflect.MakeFunc(reflect.FuncOf(typArgs, nil, false), func(a []reflect.Value) (results []reflect.Value) {
		// reconstruct input struct
		vr := reflect.New(typ)
		t := a[0].Interface().(*testing.T)
		inp := typArgs[1].String()
		for i := 0; i < typ.NumField(); i++ {
			x := a[i+2]
			field := vr.Elem().Field(i)
			if isUnsupported(field.Kind()) {
				fn, has := funcs[int(x.Int())]
				if has {
					field.Set(reflect.ValueOf(fn))
				}
			} else {
				field.Set(x)
			}
		}

		params := vr.Elem().Interface()

		// run through both converters
		fwdBuffer := &bytes.Buffer{}
		fwdWriter := testFunc(t, fwdBuffer, params)
		if _, err := fwdWriter.Write([]byte(inp)); err != nil {
			return []reflect.Value{}
		}
		gotFwd := fwdBuffer.String()

		bufWriter := testFunc(t, nil, params)
		if _, err := bufWriter.Write([]byte(inp)); err != nil {
			return []reflect.Value{}
		}

		gotBuf := bufWriter.String()

		if gotFwd != gotBuf {
			t.Errorf("forward:\n\n`%s`\n\nbuf:\n\n`%s`",
				strconv.Quote(gotFwd),
				strconv.Quote(gotBuf))
		}

		return []reflect.Value{}
	})

	t.Fuzz(f.Interface())

}

func isUnsupported(k reflect.Kind) bool {
	switch k {
	case reflect.Func:
		return true
	}
	return false
}
