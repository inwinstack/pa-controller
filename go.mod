module github.com/inwinstack/pa-controller

go 1.12

require (
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/inwinstack/blended v0.7.0
	github.com/inwinstack/pango v0.3.1
	github.com/spf13/pflag v1.0.1
	github.com/stretchr/testify v1.2.2
	github.com/thoas/go-funk v0.4.0
	k8s.io/api v0.0.0-20190620084959-7cf5895f2711
	k8s.io/apiextensions-apiserver v0.0.0-20190620085554-14e95df34f1f
	k8s.io/apimachinery v0.0.0-20190612205821-1799e75a0719
	k8s.io/client-go v0.0.0-20190620085101-78d2af792bab
)

replace (
	k8s.io/api => k8s.io/api v0.0.0-20190620084959-7cf5895f2711
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190620085554-14e95df34f1f
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190612205821-1799e75a0719
	k8s.io/client-go => k8s.io/client-go v0.0.0-20190620085101-78d2af792bab
)
