module github.com/sarathsp06/sparrow/satellites/recipes

go 1.27.3

require (
	github.com/sarathsp06/sparrow/pkg/template v0.0.0-00010101000000-000000000000
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)

replace github.com/sarathsp06/sparrow/pkg/template => ../../pkg/template
