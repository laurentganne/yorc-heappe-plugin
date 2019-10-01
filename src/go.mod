module github.com/laurentganne/heappe-plugin/v1

require (
	github.com/hashicorp/go-hclog v0.8.0
	github.com/hashicorp/go-plugin v1.0.0
	github.com/pkg/errors v0.8.0
	github.com/ystia/yorc/v4 v4.0.0-M4
	gopkg.in/cookieo9/resources-go.v2 v2.0.0-20150225115733-d27c04069d0d
)

// Due to this capital letter thing we have troubles and we have to replace it explicitly
replace github.com/Sirupsen/logrus => github.com/sirupsen/logrus v1.4.2

go 1.13
