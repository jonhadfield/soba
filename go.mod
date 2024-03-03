module github.com/jonhadfield/soba

go 1.21

toolchain go1.21.0

require (
	github.com/carlescere/scheduler v0.0.0-20170109141437-ee74d2f83d82
	github.com/hashicorp/go-retryablehttp v0.7.5
	github.com/jonhadfield/githosts-utils v0.0.0-20240227215907-fdbfc9a27143
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.9.0
	gitlab.com/tozd/go/errors v0.8.1
	gopkg.in/h2non/gock.v1 v1.1.2
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/h2non/parth v0.0.0-20190131123155-b4df798d6542 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/peterhellberg/link v1.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

//replace github.com/jonhadfield/githosts-utils => ../githosts-utils
