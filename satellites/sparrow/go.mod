module github.com/sarathsp06/sparrow/satellites/sparrow

go 1.27.2

require (
	github.com/modelcontextprotocol/go-sdk v1.8.0
	github.com/sarathsp06/sparrow/pkg/receivercode v0.0.0-00010101000000-000000000000
	github.com/sarathsp06/sparrow/pkg/signature v0.0.0-00010101000000-000000000000
	github.com/sarathsp06/sparrow/pkg/template v0.0.0-00010101000000-000000000000
	github.com/sarathsp06/sparrow/satellites/recipes v0.0.0-00010101000000-000000000000
	github.com/spf13/cobra v1.10.2
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/google/jsonschema-go v0.4.3 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/segmentio/asm v1.1.3 // indirect
	github.com/segmentio/encoding v0.5.4 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	golang.org/x/oauth2 v0.37.0 // indirect
	golang.org/x/sync v0.24.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	golang.org/x/time v0.15.0 // indirect
	golang.org/x/tools v0.51.0 // indirect
)

replace github.com/sarathsp06/sparrow/pkg/receivercode => ../../pkg/receivercode

replace github.com/sarathsp06/sparrow/pkg/signature => ../../pkg/signature

replace github.com/sarathsp06/sparrow/pkg/template => ../../pkg/template

replace github.com/sarathsp06/sparrow/satellites/recipes => ../recipes
