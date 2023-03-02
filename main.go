package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/parser"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

const usage = `Usage of dataclass:
  dataclass [flags] -type T -field F

T is the interface name.
F is the list of "fieldName typeName".

Environment variables:
  DATACLASS_DEBUG
    If set, enable debug logs.
  DATACLASS_STDOUT
    If set, write result to stdout.

Flags:`

func Usage() {
	fmt.Fprintln(os.Stderr, usage)
	flag.PrintDefaults()
}

var debugf = func(format string, v ...any) {}

func main() {
	var (
		typeName   = flag.String("type", "", "interface name; must be set")
		fieldNames = flag.String("field", "", "list of field names separated by '|'; must be set")
		goImports  = flag.String("goimports", "goimports", "goimports executable")
		output     = flag.String("output", "", "output file name; default srcdir/dataclass.go")

		redirectToStdout = os.Getenv("DATACLASS_STDOUT") != ""
		debug            = os.Getenv("DATACLASS_DEBUG") != ""
	)

	if debug {
		debugf = log.Printf
	}

	log.SetFlags(0)
	log.SetPrefix("dataclass: ")
	flag.Usage = Usage
	flag.Parse()

	validateTypeName(*typeName)

	g := newGenerator(*typeName)
	g.parsePackage(flag.Args())
	g.parseFields(*fieldNames)

	g.printf("// Code generated by \"dataclass %s\"; DO NOT EDIT.\n\n", strings.Join(os.Args[1:], " "))
	g.printf("package %s\n\n", g.pkgName)

	g.generate()

	writeResult := func(src []byte, args []string) error {
		if redirectToStdout {
			return writeResultToStdout(src, *goImports)
		}
		return writeResultToDestfile(src, *output, args, *goImports)
	}

	if err := writeResult(g.bytes(), flag.Args()); err != nil {
		log.Panic(err)
	}
}

func writeResultToDestfile(src []byte, output string, args []string, goImports string) error {
	return writeResultAndFormat(src, destFilename(output, args), goImports)
}

func writeResultToStdout(src []byte, goImports string) error {
	f, err := os.CreateTemp("", "dataclass")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(f.Name())

	if err := writeResultAndFormat(src, f.Name(), goImports); err != nil {
		return err
	}
	if _, err := f.Seek(0, os.SEEK_SET); err != nil {
		return err
	}
	if _, err := io.Copy(os.Stdout, f); err != nil {
		return err
	}
	return nil
}

func writeResultAndFormat(src []byte, fileName, goImports string) error {
	if err := os.WriteFile(fileName, src, 0600); err != nil {
		return fmt.Errorf("failed to write to %s: %w", fileName, err)
	}
	gi := &goImporter{
		goImports:  goImports,
		targetFile: fileName,
	}
	if err := gi.doImport(); err != nil {
		return fmt.Errorf("failed to goimport: %w", err)
	}
	return nil
}

func destFilename(output string, args []string) string {
	if output != "" {
		return output
	}
	return filepath.Join(destDir(args), "dataclass.go")
}

func destDir(args []string) string {
	if len(args) == 0 {
		args = []string{"."}
	}
	if len(args) == 1 && isDirectory(args[0]) {
		return args[0]
	}
	return filepath.Dir(args[0])
}

func isDirectory(p string) bool {
	x, err := os.Stat(p)
	if err != nil {
		log.Fatal(err)
	}
	return x.IsDir()
}

func validateTypeName(typeName string) {
	if typeName == "" {
		log.Panic("type must be set")
	}
	if capitalize(typeName) != typeName {
		log.Panicf("type must be public: %s", typeName)
	}
}

type goImporter struct {
	goImports  string
	targetFile string
}

func (s *goImporter) doImport() error {
	cmd := exec.Command(s.goImports, "-w", s.targetFile)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

type generator struct {
	buf          bytes.Buffer
	pkgName      string
	ifaceType    string
	rawFieldList []*rawField
}

func newGenerator(ifaceType string) *generator {
	var buf bytes.Buffer
	return &generator{
		buf:       buf,
		ifaceType: ifaceType,
	}
}

func (g *generator) printf(format string, v ...any) { fmt.Fprintf(&g.buf, format, v...) }
func (g *generator) bytes() []byte                  { return g.buf.Bytes() }

func (g *generator) parsePackage(patterns []string) {
	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedName,
	}, patterns...)
	if err != nil {
		log.Fatal(err)
	}
	if len(pkgs) != 1 {
		log.Fatalf("%d packages found", len(pkgs))
	}
	g.pkgName = pkgs[0].Name
}

func (g *generator) parseFields(fieldNameString string) {
	var (
		fields       []*rawField
		fieldNameSet = map[string]bool{}
	)
	for i, s := range strings.Split(fieldNameString, "|") {
		debugf("Parse field[%d]: %s", i, s)
		xs := strings.SplitN(s, " ", 2)
		if len(xs) != 2 {
			log.Panicf("invalid field: %s", s)
		}
		name := xs[0]
		// - cannot contain spaces
		// - must be public
		if strings.TrimSpace(name) != name || capitalize(name) != name {
			log.Panicf("invalid field name: %s", name)
		}
		typeName := xs[1]
		// validate typename
		if _, err := parser.ParseExpr(typeName); err != nil {
			log.Panicf("failed to parse field %s: %v", s, err)
		}

		debugf("Parse field[%d]: %s -> name = %s typeName = %s", i, s, name, typeName)
		fields = append(fields, &rawField{
			name:     name,
			typeName: typeName,
		})
		if fieldNameSet[name] {
			log.Panicf("invalid field duplicated name: %s", name)
		}
		fieldNameSet[name] = true
	}
	if len(fields) == 0 {
		log.Panic("no fields found")
	}
	g.rawFieldList = fields
}

type rawField struct {
	name     string
	typeName string
}

func (g *generator) generate() {
	it := newIfaceType(g.ifaceType)
	st := newStructType(decapitalize(g.ifaceType)) // private
	for _, rf := range g.rawFieldList {
		it.add(rf.name, rf.typeName)
		st.add(rf.name, rf.typeName)
	}
	g.printf(it.generate())
	g.printf(st.generate(g.ifaceType))
}

type stringBuilder struct {
	strings.Builder
}

func (s *stringBuilder) write(format string, v ...any) {
	s.WriteString(fmt.Sprintf("%s\n", fmt.Sprintf(format, v...)))
}

func capitalize(v string) string {
	return fmt.Sprintf("%s%s", strings.ToUpper(string(v[0])), v[1:])
}

func decapitalize(v string) string {
	return fmt.Sprintf("%s%s", strings.ToLower(string(v[0])), v[1:])
}

type ifaceType struct {
	name       string
	methodList []*ifaceMethod
}

func newIfaceType(name string) *ifaceType {
	return &ifaceType{
		name: name,
	}
}

func (it *ifaceType) generate() string {
	var b stringBuilder
	b.write("type %s interface {", it.name)
	for _, m := range it.methodList {
		b.write(m.generate())
	}
	b.write("}")
	return b.String()
}

func (it *ifaceType) add(itemName, itemType string) {
	it.methodList = append(it.methodList, &ifaceMethod{
		name:    itemName,
		retType: itemType,
	})
}

type ifaceMethod struct {
	name    string
	retType string
}

func (im *ifaceMethod) generate() string { return fmt.Sprintf("%s() %s", im.name, im.retType) }

type structType struct {
	name       string
	fieldList  []*structField
	methodList []*structMethod
}

func newStructType(name string) *structType {
	return &structType{
		name: name,
	}
}

func (st *structType) add(itemName, itemType string) {
	st.fieldList = append(st.fieldList, &structField{
		name:     decapitalize(itemName), // private
		typeName: itemType,
	})
	st.methodList = append(st.methodList, &structMethod{
		name:      capitalize(itemName),   // public
		fieldName: decapitalize(itemName), // private
		retType:   itemType,
	})
}

func (st *structType) generate(ifaceType string) string {
	var b stringBuilder
	b.write("type %s struct {", st.name)
	for _, f := range st.fieldList {
		b.write(f.generate())
	}
	b.write("}")
	for _, m := range st.methodList {
		b.write(m.generate(st.name))
	}
	b.write(st.generateConstructor(ifaceType))
	return b.String()
}

func (st *structType) generateConstructor(ifaceType string) string {
	var b stringBuilder
	b.write("func New%s(", ifaceType)
	for _, f := range st.fieldList {
		b.write("%s %s,", f.name, f.typeName)
	}
	b.write(") %s {", ifaceType)
	b.write("return &%s{", st.name)
	for _, f := range st.fieldList {
		b.write("%[1]s: %[1]s,", f.name)
	}
	b.write("}") // struct
	b.write("}") // func
	return b.String()
}

type structField struct {
	name     string
	typeName string
}

func (sf *structField) generate() string {
	return fmt.Sprintf("%s %s", sf.name, sf.typeName)
}

type structMethod struct {
	name      string
	fieldName string
	retType   string
}

func (sm *structMethod) generate(recvType string) string {
	return fmt.Sprintf("func (s *%s) %s() %s { return s.%s }", recvType, sm.name, sm.retType, sm.fieldName)
}
