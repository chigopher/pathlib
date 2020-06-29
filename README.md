pathlib
========

[![Build Status](https://travis-ci.org/chigopher/pathlib.svg?branch=master)](https://travis-ci.org/chigopher/pathlib) ![GitHub release (latest by date)](https://img.shields.io/github/v/release/chigopher/pathlib?style=flat-square) [![Codecov](https://img.shields.io/codecov/c/github/chigopher/pathlib?style=flat-square)](https://codecov.io/gh/chigopher/pathlib) ![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/chigopher/pathlib?style=flat-square)

Inspired by Python's pathlib, made better by Golang.

`pathlib` is an "object-oriented" package for manipulating filesystem path objects. It takes many cues from [Python's pathlib](https://docs.python.org/3/library/pathlib.html), although it does not strictly adhere to its design philosophy. It provides a simple, intuitive, easy, and abstracted interface for dealing with many different types of filesystems.

`pathlib` is currently in the alpha stage of development, meaning the API is subject to change. However, the current state of the project is already proving to be highly useful.

Examples
---------

### OsFs

Beacuse `pathlib` treats `afero` filesystems as first-class citizens, you can instantiate a `Path` object with the filesystem of your choosing.

#### Code

```go
package main

import (
	"fmt"
	"os"

	"github.com/chigopher/pathlib"
	"github.com/spf13/afero"
)

func main() {
	// Create a path on your regular OS filesystem
	path := pathlib.NewPathAfero("/home/ltclipp", afero.NewOsFs())

	subdirs, err := path.ReadDir()
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	for _, dir := range subdirs {
		fmt.Println(dir.Name())
	}
}
```

#### Output

```bash
[ltclipp@landon-virtualbox examples]$ go build .
[ltclipp@landon-virtualbox examples]$ ./examples | tail
Music
Pictures
Public
Templates
Videos
git
go
mockery_test
snap
software
```

### In-memory FS

#### Code
```go
package main

import (
	"fmt"
	"os"

	"github.com/chigopher/pathlib"
	"github.com/spf13/afero"
)

func main() {
	// Create a path using an in-memory filesystem
	path := pathlib.NewPathAfero("/", afero.NewMemMapFs())
	hello := path.Join("hello_world.txt")
	hello.WriteFile([]byte("hello world!"), 0o644)

	subpaths, err := path.ReadDir()
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	for _, subpath := range subpaths {
		fmt.Printf("Name: %s Mode: %o Size: %d\n", subpath.Name(), subpath.Mode(), subpath.Size())
	}

	bytes, _ := hello.ReadFile()
	fmt.Println(string(bytes))
}
```

#### Output

```bash
[ltclipp@landon-virtualbox examples]$ go build
[ltclipp@landon-virtualbox examples]$ ./examples 
Name: hello_world.txt Mode: 644 Size: 12
hello world!
```

Frequently Asked Questions
--------------------------

#### Why `pathlib` and not [`filepath`](https://golang.org/pkg/path/filepath/)?

[`filepath`](https://golang.org/pkg/path/filepath/) is a package that is tightly coupled to the OS filesystem APIs and also is not written in an object-oriented way. `pathlib` uses [`afero`](https://github.com/spf13/afero) under the hood for its abstracted filesystem interface, which allows you to represent a vast array of different filesystems (e.g. SFTP, HTTP, in-memory, and of course OS filesystems) using the same `Path` object.

#### Why not use `afero` directly? 

You certainly could, however `afero` does not represent a _filesystem object_ in an object-oriented way. It is only object-oriented with respect to the filesystem itself. `pathlib` is simply a thin layer on top of `afero` that provides the filesystem-object-orientation.
