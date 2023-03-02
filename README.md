# dataclass

Generate a data container.

Given

``` go
package example
// ...
```

run `dataclass -type Argument -field "Size int|Handler flag.ErrorHandling|Target string"` then generate

``` go
package example

import "flag"

type Argument interface {
	Size() int
	Handler() flag.ErrorHandling
	Target() string
}
type argument struct {
	size    int
	handler flag.ErrorHandling
	target  string
}

func (s *argument) Size() int                   { return s.size }
func (s *argument) Handler() flag.ErrorHandling { return s.handler }
func (s *argument) Target() string              { return s.target }
func NewArgument(
	size int,
	handler flag.ErrorHandling,
	target string,
) Argument {
	return &argument{
		size:    size,
		handler: handler,
		target:  target,
	}
}
```

in dataclass.go in the same directory.

# Requirements

- [goimports](https://pkg.go.dev/golang.org/x/tools/cmd/goimports)
