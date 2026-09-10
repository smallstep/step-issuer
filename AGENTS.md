# AGENTS.md

Guidance for AI coding agents working in this repository. Claude Code loads it through the one-line `@AGENTS.md` import in `CLAUDE.md`.

## Overview

step-issuer is a [cert-manager](https://cert-manager.io) external issuer for Kubernetes. It watches
cert-manager `CertificateRequest` resources whose `issuerRef.group` is `certmanager.step.sm` and signs
them against a [step-ca](https://github.com/smallstep/certificates) server using a JWK provisioner.
It defines two CRDs, `StepIssuer` (namespaced) and `StepClusterIssuer` (cluster-scoped), in API group
`certmanager.step.sm/v1beta1`. Built with kubebuilder v2 scaffolding on controller-runtime; the
step-ca client comes from `github.com/smallstep/certificates/ca`. Module: `github.com/smallstep/step-issuer`.
Binary name is `manager`; the container image is `smallstep/step-issuer`.

## Commands

```bash
make build        # make generate + CGO_ENABLED=0 go build -> bin/manager
make test         # go test ./api/... ./controllers/... -coverprofile cover.out
make lint         # golangci-lint (config fetched from smallstep/workflows) + govulncheck
make generate     # controller-gen: deepcopy (api/) + CRDs/RBAC/webhook manifests (config/)
make manifests    # CRD/RBAC/webhook manifests only
make run          # make generate + go run ./main.go against current kubeconfig
make install      # kubectl apply CRDs from config/crd/bases
make deploy       # CRDs + kustomize build config/default | kubectl apply
make bootstrap    # install golangci-lint, govulncheck, gotestsum
make clean        # rm bin/manager
```

```bash
go test -run TestResolveProvisionerPasswordFile ./controllers/   # single test
go build ./... && go vet ./...                                    # quick check, no controller-gen
```

Notes:
- `make build` and `make run` depend on `generate`, which needs `controller-gen`; if it is not on
  `PATH` the Makefile runs `go install sigs.k8s.io/controller-tools/cmd/controller-gen` (version pinned
  via `tools.go` / `go.mod`). Regeneration rewrites `api/v1beta1/zz_generated.deepcopy.go` and
  `config/crd/bases/*.yaml`; check `git status` before committing.
- `make test` covers only `./api/...` and `./controllers/...`. CI runs `gotestsum -- ./...` (all
  packages, stable and oldstable Go), `V=1 make build`, golangci-lint, govulncheck, and CodeQL via
  the shared `smallstep/workflows` `goCI.yml`. Run `go test ./...` locally to match CI.
- `make fmt` references an undefined `$(SRC)` variable and is a no-op; run `goimports -w .` directly.
- Every `make` invocation prints `./.version.sh: Command not found`; the script does not exist in the
  repo. It is harmless (`VERSION` falls back to `git describe`), so ignore it.
- No Docker, database, or cluster is needed for build or unit tests. `make run`/`install`/`deploy`
  need a kubeconfig (the README describes a kind setup with cert-manager installed).

## Generated Code - Do Not Edit

| Pattern | Generator |
|---------|-----------|
| `api/v1beta1/zz_generated.deepcopy.go` | controller-gen `object` (directive in `api/v1beta1/generate.go`, header from `hack/boilerplate.go.txt`) |
| `config/crd/bases/certmanager.step.sm_*.yaml` | controller-gen `crd` from `+kubebuilder:` markers in `api/v1beta1/*_types.go` |
| `config/rbac/role.yaml` | controller-gen `rbac:roleName=manager-role` from `+kubebuilder:rbac` markers in `controllers/` |

After changing a `*_types.go` file or an RBAC marker, run `make generate` and commit the results.

## Architecture

```
step-issuer/
├── main.go                          # flag parsing, scheme registration, manager + 3 reconcilers
├── api/v1beta1/                     # CRD Go types (StepIssuer, StepClusterIssuer), scheme, deepcopy
├── controllers/
│   ├── stepissuer_controller.go        # validates StepIssuer spec, resolves password, builds provisioner, sets Ready
│   ├── stepclusterissuer_controller.go # same for StepClusterIssuer
│   ├── certificaterequest_controller.go# signs cert-manager CertificateRequests via the cached provisioner
│   ├── password.go                     # provisioner password from Secret, env var, or file (exactly one)
│   ├── password_test.go                # the only unit tests in the repo
│   └── step_status_*reconciler.go      # Ready-condition helpers + event recording
├── provisioners/step.go             # step-ca JWK provisioner wrapper; global sync.Map cache keyed by NamespacedName
├── config/                          # kustomize tree: crd/, rbac/, manager/, webhook/, certmanager/, default/, samples/
├── docker/Dockerfile                # multi-stage alpine build, runs `make bin/manager`
├── hack/boilerplate.go.txt          # license header for generated code
├── PROJECT                          # kubebuilder project file (domain step.sm, group certmanager)
└── tools.go                         # pins controller-gen in go.mod
```

### Request flow

1. A `StepIssuer`/`StepClusterIssuer` is created. Its reconciler validates the spec (`url`, `caBundle`,
   `provisioner.name`, `provisioner.kid`, and exactly one of `passwordRef`/`passwordEnv`/`passwordFile`),
   loads the password, calls `ca.NewProvisioner` (which fetches and decrypts the JWK from step-ca), stores
   the provisioner in `provisioners.collection`, and sets `status.conditions[Ready]=True`.
2. cert-manager creates a `CertificateRequest`. `CertificateRequestReconciler` ignores it unless
   `issuerRef.group` is `certmanager.step.sm`, waits for the Approved condition (unless
   `-disable-approval-check`), skips already-issued requests, then requires the referenced issuer to be
   Ready and present in the cache.
3. `Step.Sign` decodes the CSR, builds a one-time token from CN + SANs, POSTs a sign request to step-ca
   with `NotAfter` from `spec.duration`, and writes the leaf+intermediate chain to `status.certificate`
   and the issuer `caBundle` to `status.ca`.

The provisioner cache is in-process only. After a controller restart, issuers must be re-reconciled
before any `CertificateRequest` can be signed; a request that arrives first fails with
"provisioner ... not found" and is retried.

## Conventions

- **Logging**: `logr` via controller-runtime's zap logger. Reconcilers carry a `Log logr.Logger`
  field; verbose diagnostics use `log.V(4)`. Zap flags (`-zap-devel`, `-zap-log-level`, ...) are bound
  in `main.go`.
- **Errors**: `fmt.Errorf` with `%w`. Reconcilers report failures by setting the `Ready` condition with a
  reason (`Validation`, `NotFound`, `Error`, `Pending`, `Failed`) and emitting a Kubernetes Event, then
  return the error so controller-runtime requeues.
- **Config**: stdlib `flag`, no env prefix. Flags: `-metrics-bind-address` (default `0` = disabled;
  manifests set `:8080`), `-enable-leader-election`, `-leader-election-id` (default
  `step-issuer-operator-lock`), `-disable-approval-check`.
- **Testing**: stdlib `testing` only, table-driven; no testify, envtest, or fake client in use.
- **RBAC** is declared as `+kubebuilder:rbac` markers on the reconcilers, not edited in YAML.
- **CRD compatibility**: `StepIssuer` and `StepClusterIssuer` specs are duplicated rather than shared;
  changes to one usually need mirroring in the other (types, validation, controller, provisioner ctor).
- `CRD_OPTIONS ?= "crd"` in the Makefile (comment: CRDs work back to Kubernetes 1.11, no conversion webhook).

## Related repos

- `github.com/smallstep/certificates` provides the `ca` client package used to talk to step-ca.
  Use a `replace` directive in `go.mod` to develop against a local checkout.
- `github.com/cert-manager/cert-manager` provides the `CertificateRequest` API types and
  `pkg/api/util` helpers for conditions and approval.
- Releases: pushing a `v*` tag runs `.github/workflows/release.yml`, which creates a GitHub release and
  pushes multi-arch images to `smallstep/step-issuer` (`:<version>`, plus `:latest` for non-rc tags).
  The Helm chart lives in `smallstep/helm-charts`.
