module github.com/jonhadfield/soba

go 1.16

require (
	github.com/carlescere/scheduler v0.0.0-20170109141437-ee74d2f83d82
	github.com/jonhadfield/githosts-utils v0.0.0-20220313174609-54b364027c38
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.7.0
)

//replace github.com/jonhadfield/githosts-utils => ../githosts-utils
