module github.com/jonhadfield/soba

go 1.22

toolchain go1.22.0

//replace github.com/jonhadfield/githosts-utils => ../githosts-utils

require (
	github.com/carlescere/scheduler v0.0.0-20170109141437-ee74d2f83d82
	github.com/hashicorp/go-retryablehttp v0.7.7
	github.com/jonhadfield/githosts-utils v0.0.0-20240729093740-0aeff8e5bf1c
	github.com/pkg/errors v0.9.1
	github.com/slack-go/slack v0.14.0
	github.com/stretchr/testify v1.9.0
	gitlab.com/tozd/go/errors v0.10.0
	gopkg.in/h2non/gock.v1 v1.1.2
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/h2non/parth v0.0.0-20190131123155-b4df798d6542 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/microsoft/azure-devops-go-api/azuredevops/v7 v7.1.0 // indirect
	github.com/peterhellberg/link v1.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
