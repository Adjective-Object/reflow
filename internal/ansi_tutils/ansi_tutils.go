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

type TestFunc = func(t testing.TB, writer io.Writer, input string, params interface{}) (string, error)

func RunTests(
	t *testing.T,
	testCases []TestCase,
	testFunc TestFunc,
) {
	for i, tc := range testCases {
		tc := tc
		i := i

		t.Run("test case "+strconv.Itoa(i), func(t *testing.T) {
			t.Parallel()
			t.Run("fwd", func(t *testing.T) {
				t.Parallel()
				w := &bytes.Buffer{}
				_, err := testFunc(t, w, tc.Input, tc.Params)
				if err != nil {
					t.Fatal(err)
				}

				got := w.String()
				if got != tc.Expected {
					t.Errorf("expected:\n\n`%s`\n\nActual Output:\n\n`%s`", strconv.Quote(tc.Expected), strconv.Quote(got))
				}
			})

			t.Run("buf", func(t *testing.T) {
				t.Parallel()
				got, err := testFunc(t, nil, tc.Input, tc.Params)
				if err != nil {
					t.Fatal(err)
				}
				if got != tc.Expected {
					t.Errorf("expected:\n\n`%s`\n\nActual Output:\n\n`%s`", strconv.Quote(tc.Expected), strconv.Quote(got))
				}
			})

			t.Run("fwd == buf", func(t *testing.T) {
				t.Parallel()
				w := &bytes.Buffer{}
				_, err := testFunc(t, w, tc.Input, tc.Params)
				if err != nil {
					t.Fatal(err)
				}

				gotFwd := w.String()
				bufResult, err := testFunc(t, nil, tc.Input, tc.Params)
				if err != nil {
					t.Fatal(err)
				}

				if gotFwd != bufResult {
					t.Errorf("forward:\n\n`%s`\n\nbuf:\n\n`%s`", strconv.Quote(gotFwd), strconv.Quote(bufResult))
				}
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

		v := vr.Elem().Interface()

		// run through both converters
		w := &bytes.Buffer{}
		_, err := testFunc(t, w, inp, v)
		if err != nil {
			return []reflect.Value{}
		}
		gotFwd := w.String()
		bufResult, err := testFunc(t, nil, inp, v)
		if err != nil {
			return []reflect.Value{}
		}

		if gotFwd != bufResult {
			t.Errorf("forward:\n\n`%s`\n\nbuf:\n\n`%s`", strconv.Quote(gotFwd), strconv.Quote(bufResult))
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
