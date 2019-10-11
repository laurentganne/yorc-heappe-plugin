module github.com/laurentganne/heappe-plugin/v1

require (
	github.com/bramvdbogaerde/go-scp v0.0.0-20191005185035-c96fe084709e
	github.com/hashicorp/consul v1.2.3
	github.com/hashicorp/go-hclog v0.8.0
	github.com/hashicorp/go-plugin v1.0.0
	github.com/pkg/errors v0.8.0
	github.com/ystia/yorc/v4 v4.0.0-M4
	golang.org/x/crypto v0.0.0-20190308221718-c2843e01d9a2
	gopkg.in/cookieo9/resources-go.v2 v2.0.0-20150225115733-d27c04069d0d
)

// Due to this capital letter thing we have troubles and we have to replace it explicitly
replace github.com/Sirupsen/logrus => github.com/sirupsen/logrus v1.4.2

go 1.13
