package main

import (
	"go/format"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGolden(t *testing.T) {
	for _, tc := range []*struct {
		title      string
		typeName   string
		fieldNames string
		output     string
	}{
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
			fieldNames: "First *http.Request,Second string",
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
	} {
		tc := tc
		t.Run(tc.title, func(t *testing.T) {
			g := newGenerator(tc.typeName)
			g.parseFields(tc.fieldNames)
			g.generate()
			got, err := format.Source(g.bytes())
			assert.Nil(t, err, "err=%#v, got=%s", err, string(g.bytes()))
			want, err := format.Source([]byte(tc.output))
			assert.Nil(t, err, "want")
			assert.Equal(t, strings.TrimRight(string(want), "\n"), strings.TrimRight(string(got), "\n"))
		})
	}
}
