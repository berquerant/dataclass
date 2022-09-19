package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestEndToEnd(t *testing.T) {
	d := newDataClass(t)
	defer d.close()

	for i, tc := range []struct {
		title      string
		fileName   string
		typeName   string
		fieldNames string
	}{
		{
			title:      "base",
			fileName:   "base.go",
			typeName:   "BaseType",
			fieldNames: "Root string,Reader *bytes.Buffer,From string,Element Element",
		},
	} {
		i := i
		tc := tc
		t.Run(tc.title, func(t *testing.T) {
			d.compileAndRun(
				t,
				i,
				tc.fileName,
				tc.typeName,
				tc.fieldNames,
			)
		})
	}
}

type dataClass struct {
	dir       string
	dataClass string
}

func newDataClass(t *testing.T) *dataClass {
	t.Helper()
	dc := &dataClass{}
	dc.init(t)
	return dc
}

func (dc *dataClass) init(t *testing.T) {
	t.Helper()
	dir, err := os.MkdirTemp("", "dataclass")
	if err != nil {
		t.Fatal(err)
	}
	dataClass := filepath.Join(dir, "dataclass")
	// build dataclass command
	if err := run("go", "build", "-o", dataClass); err != nil {
		t.Fatal(err)
	}
	dc.dir = dir
	dc.dataClass = dataClass
}

func (dc *dataClass) compileAndRun(
	t *testing.T,
	caseNumber int,
	fileName,
	typeName,
	fieldNames string,
) {
	t.Helper()
	src := filepath.Join(dc.dir, fileName)
	if err := copyFile(src, filepath.Join("testdata", fileName)); err != nil {
		t.Fatal(err)
	}
	dataClassSrc := filepath.Join(dc.dir, fmt.Sprintf("dataclass%d.go", caseNumber))
	if err := run(
		dc.dataClass,
		"-type", typeName,
		"-field", fieldNames,
		"-output", dataClassSrc,
	); err != nil {
		t.Fatal(err)
	}
	if err := run("go", "run", dataClassSrc, src); err != nil {
		t.Fatal(err)
	}
}

func (dc *dataClass) close() {
	os.RemoveAll(dc.dir)
}

func run(name string, arg ...string) error {
	cmd := exec.Command(name, arg...)
	cmd.Dir = "."
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func copyFile(to, from string) error {
	toFile, err := os.Create(to)
	if err != nil {
		return err
	}
	defer toFile.Close()
	fromFile, err := os.Open(from)
	if err != nil {
		return err
	}
	defer fromFile.Close()
	_, err = io.Copy(toFile, fromFile)
	return err
}
