module github.com/sarathsp06/sparrow/satellites/recipes

go 1.26.1

require gopkg.in/yaml.v3 v3.0.1

require github.com/sarathsp06/sparrow/pkg/template v0.0.0-00010101000000-000000000000

require github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect

replace github.com/sarathsp06/sparrow/pkg/template => ../../pkg/template
