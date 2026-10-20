module github.com/sarathsp06/sparrow/pkg/receivercode

go 1.27.2

require (
	github.com/sarathsp06/sparrow/pkg/signature v0.0.0-00010101000000-000000000000
	github.com/sarathsp06/sparrow/pkg/template v0.0.0-00010101000000-000000000000
)

require github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect

replace (
	github.com/sarathsp06/sparrow/pkg/signature => ../signature
	github.com/sarathsp06/sparrow/pkg/template => ../template
)
