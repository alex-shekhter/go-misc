/*
go-lazy generates implementations of lazy evaluation for types.

When no types are given, it generates implementations of all builtin types and
for interface{} (for use in merovius.de/go-misc/lazy). The output should go in
a separate file. The created code will contain a function per type that can be
used to generate a lazily evaluated version of that type. See
merovius.de/go-misc/lazy for details on the usage of this function.

The generated implementations are concurrency-safe and have a reasonably low
overhead.

The CLI is still not entirely finalized, it may be subject to change for now.

Usage:
	go-lazy [flags] [<name> <type> ...]
You must pass an even number of arguments. For each wrapped type you need to
give the name of the function and the type you want to wrap it.

The flags are:
	-package pkg
		what package the generated file should reside in. Defaults to "lazy".

	-out file
		output file, defaults to stdout.
*/
package main

import (
	"bytes"
	"flag"
	"go/format"
	"io"
	"log"
	"os"
	"text/template"
)

var implTemplate = template.Must(template.New("lazy.go").Parse(`
// This file is automatically generated by merovius.de/go-misc/cmdgo-lazy.

package {{ .Package }}

import (
	"sync"
	"sync/atomic"
)

{{ range .Types }}
	{{ template "impl" . }}
{{ end }}
`))

var _ = template.Must(implTemplate.New("impl").Parse(`
// lazy{{ .Name }} implements lazy evaluation for {{ .Type }}.
type lazy{{ .Name }} struct {
	v {{ .Type }}
	f func() {{ .Type }}
	m sync.Mutex
	o uint32
}

func (v *lazy{{ .Name }}) Get() {{ .Type }} {
	if atomic.LoadUint32(&v.o) == 0 {
		return v.v
	}

	v.m.Lock()
	defer v.m.Unlock()

	if v.o == 0 {
		v.v = v.f()
		v.o = 1
		v.f = nil
	}
	return v.v
}

// {{ .Name }} provides lazy evaluation for {{ .Type }}. f is called exactly
// once, when the result is first used.
func {{ .Name }} (f func() {{ .Type }}) func() {{ .Type }} {
	return (&lazy{{ .Name}}{f:f}).Get
}
`))

type pkg struct {
	Package string
	Types   []typ
}

type typ struct {
	Name string
	Type string
}

var defaultTypes = []typ{
	{Name: "Bool", Type: "bool"},
	{Name: "Byte", Type: "byte"},
	{Name: "Complex64", Type: "complex64"},
	{Name: "Complex128", Type: "complex128"},
	{Name: "Float32", Type: "float32"},
	{Name: "Float64", Type: "float64"},
	{Name: "Error", Type: "error"},
	{Name: "Int", Type: "int"},
	{Name: "Int8", Type: "int8"},
	{Name: "Int16", Type: "int16"},
	{Name: "Int32", Type: "int32"},
	{Name: "Int64", Type: "int64"},
	{Name: "Interface", Type: "interface{}"},
	{Name: "Rune", Type: "rune"},
	{Name: "String", Type: "string"},
	{Name: "Uint", Type: "uint"},
	{Name: "Uint8", Type: "uint8"},
	{Name: "Uint16", Type: "uint16"},
	{Name: "Uint32", Type: "uint32"},
	{Name: "Uint64", Type: "uint64"},
	{Name: "Uintptr", Type: "uintptr"},
}

var (
	pkgName = flag.String("package", "lazy", "Package the file should be in")
	outFile = flag.String("out", "", "Where to write the output (defaults to stdout)")
)

func main() {
	flag.Parse()
	log.SetFlags(0)

	var out io.Writer
	if *outFile == "" {
		out = os.Stdout
	} else {
		f, err := os.Create(*outFile)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		out = f
	}

	if flag.NArg()%2 != 0 {
		log.Fatal("Usage: go-lazy [-package=<pkg>] [<name> <type>]...")
	}

	var types []typ
	for i := 0; i < flag.NArg(); i += 2 {
		types = append(types, typ{Name: flag.Arg(i), Type: flag.Arg(i + 1)})
	}
	if len(types) == 0 {
		types = defaultTypes
	}

	buf := new(bytes.Buffer)

	if err := implTemplate.Execute(buf, pkg{Package: *pkgName, Types: types}); err != nil {
		log.Fatal(err)
	}

	output, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatal(err)
	}

	if _, err := out.Write(output); err != nil {
		log.Fatal(err)
	}
}
