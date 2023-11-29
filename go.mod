module github.com/chigopher/pathlib

go 1.18

require (
	github.com/spf13/afero v1.4.0
	github.com/stretchr/testify v1.6.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/kr/pretty v0.1.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/objx v0.1.1 // indirect
	golang.org/x/text v0.3.8 // indirect
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
	gopkg.in/yaml.v3 v3.0.0 // indirect
)

retract (
	v1.0.1
	v1.0.0 // Published accidentally
)
