module github.com/douglasmakey/admissioncontroller

replace github.com/go-logr/logr v0.1.0 => github.com/go-logr/logr v0.2.0

go 1.16

require (
	k8s.io/api v0.20.4
	k8s.io/apimachinery v0.20.4
	k8s.io/klog/v2 v2.4.0 // indirect
)
