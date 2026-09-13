module github.com/sarathsp06/sparrow/satellites/sparrow

go 1.26.1

require (
	github.com/sarathsp06/sparrow/pkg/signature v0.0.0-00010101000000-000000000000
	github.com/sarathsp06/sparrow/pkg/template v0.0.0-00010101000000-000000000000
	github.com/spf13/cobra v1.10.2
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
)

replace github.com/sarathsp06/sparrow/pkg/signature => ../../pkg/signature

replace github.com/sarathsp06/sparrow/pkg/template => ../../pkg/template
