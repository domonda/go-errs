module github.com/domonda/go-errs/cmd/go-errs-wrap

go 1.26.0

replace github.com/domonda/go-errs => ../..

require github.com/domonda/go-errs v0.0.0-00010101000000-000000000000 // replaced

require (
	github.com/stretchr/testify v1.12.1
	github.com/ungerik/go-astvisit v0.0.0-20261013080339-68c5a526b805
	golang.org/x/tools v0.52.0
)

require (
	github.com/domonda/go-pretty v1.0.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/mod v0.42.0 // indirect
	golang.org/x/sync v0.24.0 // indirect
)
