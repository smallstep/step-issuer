module github.com/smallstep/step-issuer

go 1.12

require (
	github.com/AndreasBriese/bbloom v0.0.0-20190306092124-e2d15f34fcf9 // indirect
	github.com/dgraph-io/badger v1.5.3 // indirect
	github.com/dgryski/go-farm v0.0.0-20190423205320-6a90982ecee2 // indirect
	github.com/dustin/go-humanize v1.0.0 // indirect
	github.com/go-chi/chi v4.0.2+incompatible // indirect
	github.com/go-logr/logr v0.1.0
	github.com/jetstack/cert-manager v0.9.1-0.20190801150227-8fa48c2148e3
	github.com/manifoldco/promptui v0.3.2 // indirect
	github.com/newrelic/go-agent v2.10.0+incompatible // indirect
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/rs/xid v1.2.1 // indirect
	github.com/samfoo/ansi v0.0.0-20160124022901-b6bd2ded7189 // indirect
	github.com/shurcooL/sanitized_anchor_name v1.0.0 // indirect
	github.com/sirupsen/logrus v1.4.2 // indirect
	github.com/smallstep/certificates v0.11.0-rc.3
	github.com/smallstep/cli v0.11.0-rc.4 // indirect
	github.com/smallstep/nosql v0.1.0 // indirect
	github.com/spf13/cobra v0.0.5 // indirect
	github.com/urfave/cli v1.21.0 // indirect
	go.etcd.io/bbolt v1.3.3 // indirect
	golang.org/x/net v0.0.0-20190620200207-3b0461eec859 // indirect
	golang.org/x/sys v0.0.0-20190626221950-04f50cda93cb // indirect
	gopkg.in/square/go-jose.v2 v2.3.1 // indirect
	k8s.io/api v0.0.0-20190718183219-b59d8169aab5
	k8s.io/apimachinery v0.0.0-20190612205821-1799e75a0719
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/utils v0.0.0-20190607212802-c55fbcfc754a
	sigs.k8s.io/controller-runtime v0.2.0-beta.4
)

replace k8s.io/api => k8s.io/api v0.0.0-20190718183219-b59d8169aab5

replace k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190612205821-1799e75a0719

replace k8s.io/client-go => k8s.io/client-go v0.0.0-20190718183610-8e956561bbf5
