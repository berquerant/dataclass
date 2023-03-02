package main

import (
	"fmt"
	"go/format"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type goldenTestcase struct {
	title      string
	typeName   string
	fieldNames string
	output     string
}

func (tc *goldenTestcase) test(t *testing.T) {
	g := newGenerator(tc.typeName)
	g.parseFields(tc.fieldNames)
	g.generate()
	got, err := format.Source(g.bytes())
	assert.Nil(t, err, "err=%#v, got=%s", err, string(g.bytes()))
	want, err := format.Source([]byte(tc.output))
	assert.Nil(t, err, "want")
	assert.Equal(t, strings.TrimRight(string(want), "\n"), strings.TrimRight(string(got), "\n"))
}

func TestGolden(t *testing.T) {
	simpleTestcaseTypeNames := []string{
		"int",
		"string",
		"[]int",
		"[1]int",
		"map[string]int",
		"chan string",
		"chan<- string",
		"<-chan string",
		"func()",
		"[][]int",
		"[]map[string]int",
		"map[string][]int",
		"chan []int",
		"func() error",
		"func(int)",
		"func(int) error",
		"func(int) (string, error)",
		"func(int, string) (map[string]int, error)",
		"*int",
		"*[]int",
		"flag.ErrorHandler",
		"chan chan map[string]int",
	}

	simpleTestcases := make([]goldenTestcase, len(simpleTestcaseTypeNames))
	for i, typeName := range simpleTestcaseTypeNames {
		simpleTestcases[i] = generateSimpleGoldenTestcase(typeName)
	}

	compositeTestcases := []goldenTestcase{
		{
			title:      "one_method",
			typeName:   "OneType",
			fieldNames: "OneField int",
			output: `type OneType interface{
  OneField() int
}
type oneType struct{
  oneField int
}
func (s *oneType) OneField() int { return s.oneField }
func NewOneType(
  oneField int,
) OneType {
  return &oneType{
    oneField: oneField,
  }
}`,
		},
		{
			title:      "pointer",
			typeName:   "PointerType",
			fieldNames: "First *http.Request",
			output: `type PointerType interface{
  First() *http.Request
}
type pointerType struct{
  first *http.Request
}
func (s *pointerType) First() *http.Request { return s.first }
func NewPointerType(
  first *http.Request,
) PointerType {
  return &pointerType{
    first: first,
  }
}`,
		},
		{
			title:      "two_types",
			typeName:   "TwoType",
			fieldNames: "First *http.Request|Second string",
			output: `type TwoType interface{
  First() *http.Request
  Second() string
}
type twoType struct{
  first *http.Request
  second string
}
func (s *twoType) First() *http.Request { return s.first }
func (s *twoType) Second() string { return s.second }
func NewTwoType(
  first *http.Request,
  second string,
) TwoType {
  return &twoType{
    first: first,
    second: second,
  }
}`,
		},
	}

	testcases := append(simpleTestcases, compositeTestcases...)

	for _, tc := range testcases {
		tc.test(t)
	}
}

func generateSimpleGoldenTestcase(fieldTypeName string) goldenTestcase {
	return goldenTestcase{
		title:      fieldTypeName,
		typeName:   "OneType",
		fieldNames: fmt.Sprintf("V %s", fieldTypeName),
		output:     fmt.Sprintf(simpleGoldenTestWantTemplate, fieldTypeName),
	}
}

const simpleGoldenTestWantTemplate = `type OneType interface{
  V() %[1]s
}
type oneType struct{
  v %[1]s
}
func (s *oneType) V() %[1]s { return s.v }
func NewOneType(
  v %[1]s,
) OneType {
  return &oneType{
    v: v,
  }
}`
