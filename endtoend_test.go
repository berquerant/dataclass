package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

type endToEndTestcase struct {
	title      string
	fileName   string
	typeName   string
	fieldNames string
}

func (tc *endToEndTestcase) test(t *testing.T, caseNumber int, d *dataClass) {
	d.compileAndRun(
		t,
		caseNumber,
		tc.fileName,
		tc.typeName,
		tc.fieldNames,
	)
}

func TestEndToEnd(t *testing.T) {
	const testdataDir = "testdata"

	d := newDataClass(t, testdataDir)
	defer d.close()

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
		"chan chan map[string]int",
	}

	for i, typeName := range simpleTestcaseTypeNames {
		i := i
		tc, err := generateSimpleEndToEndTestcase(
			testdataDir,
			fmt.Sprintf("simple_%d", i),
			typeName,
		)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(tc.title, func(t *testing.T) {
			tc.test(t, i, d)
			tc.close()
		})
	}

	compositeTestcases := []endToEndTestcase{
		{
			title:      "base",
			fileName:   "base.go",
			typeName:   "BaseType",
			fieldNames: "Root string|Reader *bytes.Buffer|From string|Element Element",
		},
	}

	for i, tc := range compositeTestcases {
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
	testdataDir string
	dir         string
	dataClass   string
}

func newDataClass(t *testing.T, testdataDir string) *dataClass {
	t.Helper()
	dc := &dataClass{
		testdataDir: testdataDir,
	}
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
	if err := copyFile(src, filepath.Join(dc.testdataDir, fileName)); err != nil {
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

type generatedEndToEndTestcase struct {
	endToEndTestcase
	close func()
}

func generateSimpleEndToEndTestcase(dir, title, fieldTypeName string) (*generatedEndToEndTestcase, error) {
	fileName := fmt.Sprintf("%s.go", title)
	filePath := filepath.Join(dir, fileName)
	f, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if _, err := fmt.Fprintf(f, simpleEndToEndTestcaseSourceTemplate, fieldTypeName, "%#v"); err != nil {
		return nil, err
	}
	return &generatedEndToEndTestcase{
		endToEndTestcase: endToEndTestcase{
			title:      title,
			fileName:   fileName,
			typeName:   "Data",
			fieldNames: fmt.Sprintf("V %s", fieldTypeName),
		},
		close: func() {
			os.Remove(filePath)
		},
	}, nil
}

const simpleEndToEndTestcaseSourceTemplate = `package main
import "fmt"
func main() {
  var v %[1]s
  data := NewData(v)
  fmt.Printf("%[2]s\n", data)
}
`
