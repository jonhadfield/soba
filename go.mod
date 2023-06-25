module github.com/jonhadfield/soba

go 1.19

require (
	github.com/carlescere/scheduler v0.0.0-20170109141437-ee74d2f83d82
	github.com/jonhadfield/githosts-utils v0.0.0-20230625135022-a9f43caf4d5c
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.8.4
	golang.org/x/exp v0.0.0-20230522175609-2e198f4a06a1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.4 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/peterhellberg/link v1.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/jonhadfield/githosts-utils => ../githosts-utils
