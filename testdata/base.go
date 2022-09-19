package main

import "bytes"

type Element int

const (
	Fire Element = iota
	Water
	Air
	Earth
)

func check(ok bool, msg string) {
	if !ok {
		panic(msg)
	}
}

func main() {
	const (
		rootVal    = "rootval"
		readerVal  = "bufstr"
		fromVal    = "reader"
		elementVal = Air
	)
	v := NewBaseType(rootVal, bytes.NewBufferString(readerVal), fromVal, elementVal)
	check(v.Root() == rootVal, "root")
	check(v.Reader().String() == readerVal, "reader")
	check(v.From() == fromVal, "from")
	check(v.Element() == elementVal, "element")
}
