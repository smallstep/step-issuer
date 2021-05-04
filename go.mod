module github.com/smallstep/step-issuer

go 1.13

require (
	github.com/go-logr/logr v0.3.0
	github.com/jetstack/cert-manager v1.3.1
	github.com/mattn/go-colorable v0.1.8 // indirect
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/smallstep/certificates v0.15.15
	k8s.io/api v0.20.2
	k8s.io/apiextensions-apiserver v0.20.2 // indirect
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v0.20.2
	k8s.io/utils v0.0.0-20210111153108-fddb29f9d009
	sigs.k8s.io/controller-runtime v0.8.3
)
