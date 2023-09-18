# Step Issuer

Step Issuer is a [Kubernetes](https://kubernetes.io/) [`cert-manager`](https://cert-manager.io) [`CertificateRequest`](https://cert-manager.io/docs/concepts/certificaterequest/) controller
that uses the [`step-ca`](https://github.com/smallstep/certificates) online Certificate Authority (CA) to sign certificate requests.
It's perfect for getting certificates for ingresses
and other Kubernetes resources
from an internal CA
instead of Let's Encrypt or another public CA.

## Installation

### Before you begin

In this guide, we assume that you have a Kubernetes environment
with `cert-manager` v1.0.0 or higher installed.

### Overview

The general install process is:
* Install `step-ca`
* Install `step-issuer`
* Generate and install `step-issuer` configuration
* Create your first certificate!

### 1. Install `step-ca`

Step Issuer uses [`step-ca`](https://github.com/smallstep/certificates) as its backing Certificate Authority (CA) for signing `CertificateRequest` resources.

#### Install `step-ca` via Helm 

The easiest way to install `step-ca` in Kubernetes is via [Helm](https://helm.sh/).

```sh
helm repo add smallstep  https://smallstep.github.io/helm-charts
helm repo update
helm install step-certificates smallstep/step-certificates
```

#### Install `step-ca` another way

Alternatively, you can [install a `step-ca` server](https://smallstep.com/docs/step-ca/installation) that resides outside of Kubernetes,
or use a [Certificate Manager](https://smallstep.com/certificate-manager/) cloud CA.

### 2. Install `step-issuer`

Finally, we need to install `step-issuer`.
The easiest way to install it is via Helm:

```sh
helm install step-issuer smallstep/step-issuer
```

**Alternatively, see [Installing from Source](#installing-from-source), below.**

### 3. Configure `step-issuer`

To connect `step-issuer` to our Certificate Authority,
we will need the following configuration information from our CA:

* **The CA's URL**
  When `step-ca` is installed via Helm, the default CA URL is:

  ```sh
  $ CA_URL=https://step-certificates.default.svc.cluster.local
  ```

* **The CA's root certificate**
    The root certificate can be obtained from the `step-certificates-certs`
    `ConfigMap`.

    To retrieve it, run:

    ```sh
    $ kubectl get -o jsonpath="{.data['root_ca\.crt']}" configmaps/step-certificates-certs
    -----BEGIN CERTIFICATE-----
    MIIBizCCATGgAwIBAgIQO+EAh8y/0V9P0XpHrVj5NTAKBggqhkjOPQQDAjAkMSIw
    IAYDVQQDExlTdGVwIENlcnRpZmljYXRlcyBSb290IENBMB4XDTE5MDgxMzE5MTUw
    MloXDTI5MDgxMDE5MTUwMlowJDEiMCAGA1UEAxMZU3RlcCBDZXJ0aWZpY2F0ZXMg
    Um9vdCBDQTBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABAMVL7W0Pm3oJUfI4wXd
    klDEnn5XSmj86X0amCA0gcO1tITPmCW3Bpe4pOoWUvZVeQdoScq7znkUt2/G2t1N
    71ijRTBDMA4GA1UdDwEB/wQEAwIBBjASBgNVHRMBAf8ECDAGAQH/AgEBMB0GA1Ud
    DgQWBBRucPrVnPvZN0r4AU9Lg2/eBrx7kjAKBggqhkjOPQQDAgNIADBFAiBRRAtk
    5zLcGhCahmPnW20dLitC3EWMiQ4lDp7aEz+EPAIhAI9fVs5qoItmT8jp6ZKU5Q2u
    aDPk8k2CnN27rFsYWupL
    -----END CERTIFICATE-----
    ```

    For `step-issuer`, we will Base64-encode this PEM block:

    ```sh
    $ CA_ROOT_B64=$(kubectl get -o jsonpath="{.data['root_ca\.crt']}" configmaps/step-certificates-certs | base64 -w 0)
    $ echo $CA_ROOT_B64
    LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJpekNDQVRHZ0F3SUJBZ0lRTytFQWg4eS8wVjlQMFhwSHJWajVOVEFLQmdncWhrak9QUVFEQWpBa01TSXcKSUFZRFZRUURFeGxUZEdWd0lFTmxjblJwWm1sallYUmxjeUJTYjI5MElFTkJNQjRYRFRFNU1EZ3hNekU1TVRVdwpNbG9YRFRJNU1EZ3hNREU1TVRVd01sb3dKREVpTUNBR0ExVUVBeE1aVTNSbGNDQkRaWEowYVdacFkyRjBaWE1nClVtOXZkQ0JEUVRCWk1CTUdCeXFHU000OUFnRUdDQ3FHU000OUF3RUhBMElBQkFNVkw3VzBQbTNvSlVmSTR3WGQKa2xERW5uNVhTbWo4NlgwYW1DQTBnY08xdElUUG1DVzNCcGU0cE9vV1V2WlZlUWRvU2NxN3pua1V0Mi9HMnQxTgo3MWlqUlRCRE1BNEdBMVVkRHdFQi93UUVBd0lCQmpBU0JnTlZIUk1CQWY4RUNEQUdBUUgvQWdFQk1CMEdBMVVkCkRnUVdCQlJ1Y1ByVm5QdlpOMHI0QVU5TGcyL2VCcng3a2pBS0JnZ3Foa2pPUFFRREFnTklBREJGQWlCUlJBdGsKNXpMY0doQ2FobVBuVzIwZExpdEMzRVdNaVE0bERwN2FFeitFUEFJaEFJOWZWczVxb0l0bVQ4anA2WktVNVEydQphRFBrOGsyQ25OMjdyRnNZV3VwTAotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==
    ```

* **CA credentials that `cert-manager` will use**
  CA credentials include an admin provisioner name, key id (`kid`), and a secret containing the provisioner password.

   The default Helm installation will configure one provisioner named `admin`:

  ```sh
  $ kubectl get -o jsonpath="{.data['ca\.json']}" configmaps/step-certificates-config | jq .authority.provisioners[0]
  {
    "type": "JWK",
    "name": "admin",
    "key": {
      "use": "sig",
      "kty": "EC",
      "kid": "MXxpNphrheA80gO5uUbGZ7_RdaXU8KrmKT0fVL181L8",
      "crv": "P-256",
      "alg": "ES256",
      "x": "Pi8WGmlN7zduPRhom9pYeVd48n_Pby3GopsEdkDPVdA",
      "y": "dvPWbf-CofXpBSW4uzEOPU5mEqfzi0QSjmLBnYc7nnA"
    },
    "encryptedKey": "eyJhbGciOiJQQkVTMi1IUzI1NitBMTI4S1ciLCJjdHkiOiJqd2sranNvbiIsImVuYyI6IkEyNTZHQ00iLCJwMmMiOjEwMDAwMCwicDJzIjoiUGpxRU1QZlJsOFU4X1dZQVFUZ1lKUSJ9.pawzDHDIr9Dmd49YYNkdSr5Ndl5EjKd1Q4Ac0O3S0dP0FDTRzn4gWA.aJmiBLNy-fEXKZDs.RZ2sob5zJ7oIUq02Btlb91M6w9P3Vr0yTVpp6CfvPRt9raieO4NmUOlGBB4Kg6_o2fYZxZFHFg9NqjfRzXusFpJVe1kF0qmVFAqhaT97_89nuncSMp9Guj-7e-hyrxp3WwIz6IDF1cqMtKIq3exWZJlqTPNtcjdPOT02qKzb3Msqp1ruifG6WrRNv34knCylq5H2bf5fOCaL_KIxWUQ-nE3KJoIqe-EgaigZtvBWdg-xO8M7SKYDsla4g1Gb0Vd23hBJl9Qg3UjlirmVqBY90dQF-cejF0JeCwQoiflqhN6c6kNjriZF4qNTkhWNnQOEhtiZQSdOoP9b9Ai6En4._QRZxj3vaGk2mJ3K2Oflog"
  }
  ```

   We recommend adding a separate provisioner for `cert-manager`,
   but for this guide we will use the default one.
   
   The provisioner kid can be obtained from the `step-certificates-config`
`ConfigMap`, or running:
   
  ```sh
  $ CA_PROVISIONER_NAME=admin
  $ CA_PROVISIONER_KID=$(kubectl get -o jsonpath="{.data['ca\.json']}" configmaps/step-certificates-config | jq -r .authority.provisioners[0].key.kid)
  $ echo $CA_PROVISIONER_KID
  MXxpNphrheA80gO5uUbGZ7_RdaXU8KrmKT0fVL181L8
  ```

  Fianlly, we need the password to decrypt the provisioner private key, this is
  available in the secret `step-certificates-provisioner-password`.
  Here's what it looks like:

  ```sh
  $ kubectl get -o jsonpath='{.data.password}' secret/step-certificates-provisioner-password | base64 --decode
  MfKmjQrR1iw3ZvTd4CImQfhwIbdq2FRp
  ```

  We won't use the plain password to configure `step-issuer`, we will be
  referencing the same secret.

To recap, we got:

* The CA url `https://step-certificates.default.svc.cluster.local`
* The root CA certificate PEM, base64-encoded
* The provisioner name `admin`
* The provisioner `kid` value
* And the provisioner password secret `step-certificates-provisioner-password`
  and key `password`

Now let's generate the `StepIssuer` resource.

```sh
$ cat <<EOF > step-issuer.yaml
---
apiVersion: certmanager.step.sm/v1beta1
kind: StepIssuer
metadata:
  name: step-issuer
  namespace: default
spec:
  # The CA URL:
  url: $CA_URL
  # The base64 encoded version of the CA root certificate in PEM format:
  caBundle: $CA_ROOT_B64
  # The provisioner name, kid, and a reference to the provisioner password secret:
  provisioner:
    name: $CA_PROVISIONER_NAME
    kid: $CA_PROVISIONER_KID
    passwordRef:
      name: step-certificates-provisioner-password
      key: password
---
EOF
```

Finally, we'll apply our configuration:

```sh
$ kubectl apply -f step-issuer.yaml
```

Moments later, the resource should be ready:

```sh
$ kubectl get stepissuers.certmanager.step.sm step-issuer -o yaml
apiVersion: certmanager.step.sm/v1beta1
kind: StepIssuer
...
status:
  conditions:
  - lastTransitionTime: "2019-08-14T00:11:22Z"
    message: StepIssuer verified and ready to sign certificates
    reason: Verified
    status: "True"
    type: Ready
```

Your `StepIssuer` is ready to sign certificates.

### 4. Create your first `Certificate`

Step Issuer has a controller watching for CertificateRequest resources, when one
is created, the controller checks that it belongs to it, looking for the group
`certmanager.step.sm`, then it loads the issuer `step-issuer` that will be in
charge the certificate.

To create a CertificateRequest we first need a CSR. We can use
[step](https://github.com/smallstep/cli) to create one, we will use the password
`my-password` to encrypt the private key:

```sh
$ step certificate create --csr internal.smallstep.com internal.csr internal.key
Please enter the password to encrypt the private key:
Your certificate signing request has been saved in internal.csr.
Your private key has been saved in internal.key.
$ cat internal.csr
-----BEGIN CERTIFICATE REQUEST-----
MIIBEDCBtwIBADAhMR8wHQYDVQQDExZpbnRlcm5hbC5zbWFsbHN0ZXAuY29tMFkw
EwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEWYaOephFfhvfSyv7hoPOpKA8IwSBBfTV
xLW3ROYGP1M5DuFE8NFSYICE2Hw7xdP9oaSy+v5Dou5KZNr53D2/4KA0MDIGCSqG
SIb3DQEJDjElMCMwIQYDVR0RBBowGIIWaW50ZXJuYWwuc21hbGxzdGVwLmNvbTAK
BggqhkjOPQQDAgNIADBFAiAqSDrJ29mK5QM2WEL5mtWVt9FZtpBWaPWUWQNuvHJl
ZAIhAP95OPGkCZnDiLxydwPiectue+c4HpUwdaaN4JmE1fyh
-----END CERTIFICATE REQUEST-----
$ cat internal.key
-----BEGIN EC PRIVATE KEY-----
Proc-Type: 4,ENCRYPTED
DEK-Info: AES-256-CBC,ad8a6717659e9ac184a900ba710b7254

RjwPP0j64ERhCT7AaOQ9UPMNsKipYwJJfmYZJQhHbHopP7aX90/Qw/GBECGk2H6G
MsSKpzGbQVk82VNf55ecgNYANVbZdQhzmOLXRiGTmoSym//mOR+AvDzSa2J174vQ
gg0xRmbSiql+jIrjqjyKvLAt5PczoEi3B2u6L3rwDpQ=
-----END EC PRIVATE KEY-----
```

If your application does not support encrypted keys, you can add the flags
`--no-password --insecure` to the previous command.

We are almost ready to create our CertificateRequest YAML, we only need to
encode using base64 our new CSR:

```sh
$ CSR=$(cat internal.csr | base64)
$ echo $CSR
LS0tLS1CRUdJTiBDRVJUSUZJQ0FURSBSRVFVRVNULS0tLS0KTUlJQkVEQ0J0d0lCQURBaE1SOHdIUVlEVlFRREV4WnBiblJsY201aGJDNXpiV0ZzYkhOMFpYQXVZMjl0TUZrdwpFd1lIS29aSXpqMENBUVlJS29aSXpqMERBUWNEUWdBRVdZYU9lcGhGZmh2ZlN5djdob1BPcEtBOEl3U0JCZlRWCnhMVzNST1lHUDFNNUR1RkU4TkZTWUlDRTJIdzd4ZFA5b2FTeSt2NURvdTVLWk5yNTNEMi80S0EwTURJR0NTcUcKU0liM0RRRUpEakVsTUNNd0lRWURWUjBSQkJvd0dJSVdhVzUwWlhKdVlXd3VjMjFoYkd4emRHVndMbU52YlRBSwpCZ2dxaGtqT1BRUURBZ05JQURCRkFpQXFTRHJKMjltSzVRTTJXRUw1bXRXVnQ5Rlp0cEJXYVBXVVdRTnV2SEpsClpBSWhBUDk1T1BHa0NabkRpTHh5ZHdQaWVjdHVlK2M0SHBVd2RhYU40Sm1FMWZ5aAotLS0tLUVORCBDRVJUSUZJQ0FURSBSRVFVRVNULS0tLS0K
```

And put everything together:

```sh
$ cat <<EOF > csr.yaml
apiVersion: cert-manager.io/v1
kind: CertificateRequest
metadata:
  name: internal-smallstep-com
  namespace: default
spec:
  # The base64 encoded version of the certificate request in PEM format.
  request: $CSR
  # The duration of the certificate
  duration: 24h
  # If the certificate will be a CA or not.
  # Step certificates won't accept a certificate request if this value is true,
  # you can also omit this.
  isCA: false
  # A reference to the issuer in charge of signing the CSR.
  issuerRef:
    group: certmanager.step.sm
    kind: StepIssuer
    name: step-issuer
EOF
```

We apply it using kubectl:

```sh
$ kubectl apply -f csr.yaml
certificaterequest.cert-manager.io/internal-smallstep-com configured
```

And moments later the bundled signed certificate with the intermediate as well
as the root certificate will be available in the resource:

```sh
$ kubectl get certificaterequests.cert-manager.io internal-smallstep-com -o yaml
apiVersion: cert-manager.io/v1
kind: CertificateRequest
...
status:
  ca: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJqRENDQVRLZ0F3SUJBZ0lSQU1GREdMMVIzR3pGdW5qOU0vaEgxaVF3Q2dZSUtvWkl6ajBFQXdJd0pERWkKTUNBR0ExVUVBeE1aUTJWeWRDQk5ZVzVoWjJWeUlGUmxjM1FnVW05dmRDQkRRVEFlRncweE9UQTJNall4TmpVMgpNemxhRncweU9UQTJNak14TmpVMk16bGFNQ1F4SWpBZ0JnTlZCQU1UR1VObGNuUWdUV0Z1WVdkbGNpQlVaWE4wCklGSnZiM1FnUTBFd1dUQVRCZ2NxaGtqT1BRSUJCZ2dxaGtqT1BRTUJCd05DQUFUYnFIUStHZzZZYzVOWlEzOHEKTkx1WWZKa2xMTHV2QjJobkRjaC9iM2ltZVFmRUFOQ2lYeE9CdG5PY0FKdVNQM3NxeE5HWlhQajZFeUppbmNBbQpFRXRUbzBVd1F6QU9CZ05WSFE4QkFmOEVCQU1DQVFZd0VnWURWUjBUQVFIL0JBZ3dCZ0VCL3dJQkFUQWRCZ05WCkhRNEVGZ1FVOTFWSmxma0RmdzB6aWFXV2ZjeE9CSkdEN3hRd0NnWUlLb1pJemowRUF3SURTQUF3UlFJZ1p3cnMKTlNMTTdWMlBNVStiZ1NZa1BmdjdBVGU1TG9zNlJJMUQrUFpIcmlVQ0lRQ0xjZ3lXa0cyaDZKNGV0Vkh4cDRSTApaVEpreDh1eElOcHJLc05od2ZqZFdnPT0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=
  certificate: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNMVENDQWRPZ0F3SUJBZ0lRZXdHWE9GSzdjaEVxT2o5SmREaGwvVEFLQmdncWhrak9QUVFEQWpBc01Tb3cKS0FZRFZRUURFeUZEWlhKMElFMWhibUZuWlhJZ1ZHVnpkQ0JKYm5SbGNtMWxaR2xoZEdVZ1EwRXdIaGNOTVRrdwpPREV5TWpNd05UVXdXaGNOTVRrd09ERXpNREF3TmpVd1dqQWJNUmt3RndZRFZRUURFeEJVWlhOMElFTnZiVzF2CmJpQk9ZVzFsTUZrd0V3WUhLb1pJemowQ0FRWUlLb1pJemowREFRY0RRZ0FFSG1zMi9kVWNyYks2WlViZktIRWYKVzIvMW5BZEJMekcxMGZHek1lamFTNmtyejJiRmxYb1FkNWhCSnYzUStNbFJhckMwZXhtUG0yNjhkeTBraThmdQpmNk9CNXpDQjVEQU9CZ05WSFE4QkFmOEVCQU1DQmFBd0hRWURWUjBsQkJZd0ZBWUlLd1lCQlFVSEF3RUdDQ3NHCkFRVUZCd01DTUIwR0ExVWREZ1FXQkJUZWxEaS90TmZ6WGJ0dEFhdHJmNGFIRnpVZDFUQWZCZ05WSFNNRUdEQVcKZ0JTS01mb2JPRTJtT0NSenhFajNaM1FRM1J6aU5UQWhCZ05WSFJFRUdqQVlnaFpwYm5SbGNtNWhiQzV6YldGcwpiSE4wWlhBdVkyOXRNRkFHRENzR0FRUUJncVJreGloQUFRUkFNRDRDQVFFRURHTmxjblF0YldGdVlXZGxjZ1FyCldqSlRMV3RWV1dWWmNrVmtSRTR6TWxKWU1IcHFiREY0V1MxWVVuUndlSFZrUXpKb2JYQnNaMHMyVlRBS0JnZ3EKaGtqT1BRUURBZ05JQURCRkFpQTNDbEdHVjlPeXYxdGlHWjBUQzNsY1JrQWVOR1ZuOWZvcllhM0tuZHc5bWdJaApBTW1iL0xEOGt3S0x2RUcrRW04bkVMa0VaWnhHeDJHclcrQXd3R2YxSVRxLwotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCi0tLS0tQkVHSU4gQ0VSVElGSUNBVEUtLS0tLQpNSUlCc3pDQ0FWcWdBd0lCQWdJUVVyQ202NFYzcWtEWjdZTlZ2RklQenpBS0JnZ3Foa2pPUFFRREFqQWtNU0l3CklBWURWUVFERXhsRFpYSjBJRTFoYm1GblpYSWdWR1Z6ZENCU2IyOTBJRU5CTUI0WERURTVNRFl5TmpFMk5UWXoKT1ZvWERUSTVNRFl5TXpFMk5UWXpPVm93TERFcU1DZ0dBMVVFQXhNaFEyVnlkQ0JOWVc1aFoyVnlJRlJsYzNRZwpTVzUwWlhKdFpXUnBZWFJsSUVOQk1Ga3dFd1lIS29aSXpqMENBUVlJS29aSXpqMERBUWNEUWdBRXdzcU05QTVuCllnYkxPZHI3YVdXMWZOQ3F6bkxZcUUrVWdka054UytNbEhpN1RZWWpITVdNVEtKdFg0ZktDRXZkWG9pdm05MGIKQm5JVkFFNjBaQVZaRktObU1HUXdEZ1lEVlIwUEFRSC9CQVFEQWdFR01CSUdBMVVkRXdFQi93UUlNQVlCQWY4QwpBUUF3SFFZRFZSME9CQllFRklveCtoczRUYVk0SkhQRVNQZG5kQkRkSE9JMU1COEdBMVVkSXdRWU1CYUFGUGRWClNaWDVBMzhOTTRtbGxuM01UZ1NSZys4VU1Bb0dDQ3FHU000OUJBTUNBMGNBTUVRQ0lBcGFHYkNHS0tYcXZGaWQKdEtoL0pBeEJSSGRQTlc5K1l1NjBvQzEreFp0NUFpQmZScmFKNlFIcmpKQnFFZWQ3ODY1ZmRYZDFsR2FKQXkyMgp4b1VRWnNvSFl3PT0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=
  conditions:
  - lastTransitionTime: "2019-08-14T00:12:45Z"
    message: Certificate issued
    reason: Issued
    status: "True"
    type: Ready
```

**Now you are ready to use the TLS certificate in your app. Happy signing 🎉**

## Notes

### Using the Certificate resource

Before supporting `CertificateRequest`,
cert-manager supported the resource `Certificate`.
This allows you to create TLS certificates providing only X.509 properties
like the common name, DNS or IP addresses SANs.
Cert Manager now provides a method to support `Certificate` resources
using `CertificateRequest` controllers
such as step-issuer.

The YAML for a `Certificate` resource looks like:

```yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: backend-smallstep-com
  namespace: default
spec:
  # The secret name to store the signed certificate
  secretName: backend-smallstep-com-tls
  # Common Name
  commonName: backend.smallstep.com
  # DNS SAN
  dnsNames:
    - localhost
    - backend.smallstep.com
  # IP Address SAN
  ipAddresses:
    - "127.0.0.1"
  # Duration of the certificate
  duration: 24h
  # Renew 8 hours before the certificate expiration
  renewBefore: 8h
  # The reference to the step issuer
  issuerRef:
    group: certmanager.step.sm
    kind: StepIssuer
    name: step-issuer
```

To apply the `Certificate` resource, run:

```sh
$ kubectl apply -f config/samples/certificate.yaml
certificates.cert-manager.io/backend-smallstep-com created
```

Moments later a CertificateRequest will be automatically created by
cert-manager:

```sh
$ kubectl get certificates.cert-manager.io
NAME                               READY   AGE
backend-smallstep-com-2152809657   True    22s
internal-smallstep-com             True    1h
```

The Step Issuer gets this CertificateRequest and sends the sign request to `step
certificates`, and stores the signed certificate in the same resource. Cert
manager gets the signed certificate and stores the signed and root certificate
as well as the generated key in the secret provided in the YAML file property
`secretName`.

```sh
$ kubectl get secrets backend-smallstep-com-tls -o yaml
apiVersion: v1
data:
  ca.crt: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJpekNDQVRHZ0F3SUJBZ0lRTytFQWg4eS8wVjlQMFhwSHJWajVOVEFLQmdncWhrak9QUVFEQWpBa01TSXcKSUFZRFZRUURFeGxUZEdWd0lFTmxjblJwWm1sallYUmxjeUJTYjI5MElFTkJNQjRYRFRFNU1EZ3hNekU1TVRVdwpNbG9YRFRJNU1EZ3hNREU1TVRVd01sb3dKREVpTUNBR0ExVUVBeE1aVTNSbGNDQkRaWEowYVdacFkyRjBaWE1nClVtOXZkQ0JEUVRCWk1CTUdCeXFHU000OUFnRUdDQ3FHU000OUF3RUhBMElBQkFNVkw3VzBQbTNvSlVmSTR3WGQKa2xERW5uNVhTbWo4NlgwYW1DQTBnY08xdElUUG1DVzNCcGU0cE9vV1V2WlZlUWRvU2NxN3pua1V0Mi9HMnQxTgo3MWlqUlRCRE1BNEdBMVVkRHdFQi93UUVBd0lCQmpBU0JnTlZIUk1CQWY4RUNEQUdBUUgvQWdFQk1CMEdBMVVkCkRnUVdCQlJ1Y1ByVm5QdlpOMHI0QVU5TGcyL2VCcng3a2pBS0JnZ3Foa2pPUFFRREFnTklBREJGQWlCUlJBdGsKNXpMY0doQ2FobVBuVzIwZExpdEMzRVdNaVE0bERwN2FFeitFUEFJaEFJOWZWczVxb0l0bVQ4anA2WktVNVEydQphRFBrOGsyQ25OMjdyRnNZV3VwTAotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==
  tls.crt: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURIRENDQXNPZ0F3SUJBZ0lRWFg2V1BBTm5ZZmhGbUQ0UGUvZmdHVEFLQmdncWhrak9QUVFEQWpBc01Tb3cKS0FZRFZRUURFeUZUZEdWd0lFTmxjblJwWm1sallYUmxjeUJKYm5SbGNtMWxaR2xoZEdVZ1EwRXdIaGNOTVRrdwpPREUwTVRrd09URTVXaGNOTVRrd09ERTFNVGt3T1RFNVdqQTNNUlV3RXdZRFZRUUtFd3hqWlhKMExXMWhibUZuClpYSXhIakFjQmdOVkJBTVRGV0poWTJ0bGJtUXVjMjFoYkd4emRHVndMbU52YlRDQ0FTSXdEUVlKS29aSWh2Y04KQVFFQkJRQURnZ0VQQURDQ0FRb0NnZ0VCQU1OcHN3T3hyNytUYmNVeU9UQ2YrVHVvVUl4WDVUaGNvMmNkWEdnZApXazJ2S0JBa0ZlU3hYTlR1VXE5d1Z1UFZaWWdmbm5xdnVqUGx0WTBzVXhoa3V5QkhQelNzYXBjSEo3ZGFZWmVNCjBLaGlLekJNV05IT2dYUnFPWTlkWnBDY2tjT21ZL216QlpnbUVMaW1xSittRVBTall3V2NtUmVYMzV4V3ZlV2MKKzEvUnovU2xYb1FYU2tuWENhdTY0VzZsQjlRYnl3UFdYQThqbklQcU43WktHekhEdlNOWit3L1pWNkNvYjlCbApiQkhialZ0MUxiZXNCNVpCeS9zMko3S0p3Q3FHMWNWZUg0R2N2UlhXT3FJZkFOUzNBSmVueHZJSWlpeE9ueWlsCldhUm1Bd0x0b2JBQzRuei9mOFpDbXJtdlVpdTB1bHFNTkR4VnZRcm9ZU1BzVUVFQ0F3RUFBYU9COERDQjdUQU8KQmdOVkhROEJBZjhFQkFNQ0JhQXdIUVlEVlIwbEJCWXdGQVlJS3dZQkJRVUhBd0VHQ0NzR0FRVUZCd01DTUIwRwpBMVVkRGdRV0JCVDNoRGZxZk96dTBCaWpkQmFzbUhaVnA1eXovekFmQmdOVkhTTUVHREFXZ0JSMFFtU0V0Q1pVCkdUM01vbFZ4NkVqenA3VWZqekF4QmdOVkhSRUVLakFvZ2hWaVlXTnJaVzVrTG5OdFlXeHNjM1JsY0M1amIyMkMKQ1d4dlkyRnNhRzl6ZEljRWZ3QUFBVEJKQmd3ckJnRUVBWUtrWk1Zb1FBRUVPVEEzQWdFQkJBVmhaRzFwYmdRcgpUalpKT1RsWmRXczNhVWRFVFd0ZlpWY3pVV0ZPTW1Ga2JVTnpja001VlhWRVRqSTNaR3hHV0ZWUGN6QUtCZ2dxCmhrak9QUVFEQWdOSEFEQkVBaUJxYmsyd1NqR1p6VDRGVXdpVTZickMzd0l4U3V2U3hCODMwcUx1YUVWTmVRSWcKU3cwUUZhTGVXdkYrYUZCdUN0TWo1ajlMMFl0TGRJdVhqZ3dvaWFNSzFVVT0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQotLS0tLUJFR0lOIENFUlRJRklDQVRFLS0tLS0KTUlJQnREQ0NBVnFnQXdJQkFnSVFEamJ2MnRRK1ZaelJzejZ6ZkFDdTBEQUtCZ2dxaGtqT1BRUURBakFrTVNJdwpJQVlEVlFRREV4bFRkR1Z3SUVObGNuUnBabWxqWVhSbGN5QlNiMjkwSUVOQk1CNFhEVEU1TURneE16RTVNVFV3Ck1sb1hEVEk1TURneE1ERTVNVFV3TWxvd0xERXFNQ2dHQTFVRUF4TWhVM1JsY0NCRFpYSjBhV1pwWTJGMFpYTWcKU1c1MFpYSnRaV1JwWVhSbElFTkJNRmt3RXdZSEtvWkl6ajBDQVFZSUtvWkl6ajBEQVFjRFFnQUVqVjVGVkUrNQppNlVjR1FZbjM1am5uSnl1Q3dWYnNYVTV3ZUdtMVVWSk04S04ydlNzMnpvbTNDN0d0SW1Va2hYd1RVbmI3cVBhCmNpQmhNUzZlQlNqTVNhTm1NR1F3RGdZRFZSMFBBUUgvQkFRREFnRUdNQklHQTFVZEV3RUIvd1FJTUFZQkFmOEMKQVFBd0hRWURWUjBPQkJZRUZIUkNaSVMwSmxRWlBjeWlWWEhvU1BPbnRSK1BNQjhHQTFVZEl3UVlNQmFBRkc1dwordFdjKzlrM1N2Z0JUMHVEYjk0R3ZIdVNNQW9HQ0NxR1NNNDlCQU1DQTBnQU1FVUNJSERPNWpkQ1oraDNhdEVlCjdKSFJLWmNNNkdGazZjUkJUSTJDcElmRW5Yc0lBaUVBK3RwRHVpM1RPdnBLWDdXbE55NlR3ZXc1UmlaYjhvUnUKN0FYOVk4UXJHdUE9Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
  tls.key: LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFcFFJQkFBS0NBUUVBdzJtekE3R3Z2NU50eFRJNU1KLzVPNmhRakZmbE9GeWpaeDFjYUIxYVRhOG9FQ1FWCjVMRmMxTzVTcjNCVzQ5VmxpQitlZXErNk0rVzFqU3hUR0dTN0lFYy9OS3hxbHdjbnQxcGhsNHpRcUdJck1FeFkKMGM2QmRHbzVqMTFta0p5Unc2WmorYk1GbUNZUXVLYW9uNllROUtOakJaeVpGNWZmbkZhOTVaejdYOUhQOUtWZQpoQmRLU2RjSnE3cmhicVVIMUJ2TEE5WmNEeU9jZytvM3Rrb2JNY085STFuN0Q5bFhvS2h2MEdWc0VkdU5XM1V0CnQ2d0hsa0hMK3pZbnNvbkFLb2JWeFY0ZmdaeTlGZFk2b2g4QTFMY0FsNmZHOGdpS0xFNmZLS1ZacEdZREF1MmgKc0FMaWZQOS94a0thdWE5U0s3UzZXb3cwUEZXOUN1aGhJK3hRUVFJREFRQUJBb0lCQVFDZWV6ZnE5QTJFQXE1UQo4c1Y5RVJEUitGU3pMWW5DWnlkQ3RvWStEaWd4dnE5d1A4UGR3Slo0UG55aXVpcE9Cc0NjWUlCb0llS1N1bWErCmdzYzFqbVJRN2xkdGdiUEVudEh3R3dYeElnd0xzK293OW9wR1JnT3BoWWovSTVIT0VKMExId1FQKzhlNnVJeHgKSlFDMjBia0lud1h0QkM4SSttd280QlNNaHY4N21vTmh0M2dNRFZiNmZsenBzdUpMaDZhc1JnN1FkbVhYQVJHVgpDRm5NNnYyWjhsbnR5cVRObWlZSjlvang0L0JoV1poTDV6RXVYMXBYcGFxZWZ3Ynl0SU4xOThRYUVDQlZNeUlsCkdXZVc0dXdrTTVzUWRBM3FQaU51N2hEcGwyQUdVK0ZEZllEVzQ4SElFZDkrUXNmaUZZZk5BM0sybTZ4VXQwZ3QKNHRWQlc0b3RBb0dCQU5KUXNVVWw0ZlhkZ0NqU2FZL3VzajV6cnpQMWdyNDJTdGxDSi85a1AxSkhvYTY5VnVCVgo5alh2dmc3bEJoQjZyWDhCNUZsNnFYYXJ3ci9uVFhwc0JwNU4yM1Y3RHZlQnpFMjNrLzlLNlE3ayt0UVdZUGgwCjJkbnZqVktqYklSaVozK0ZnUUU0ZW90ekhHYzNKS3FKRCsySUF1clczZ3RuQk1QQTA1bEdGbkl6QW9HQkFPM2MKVDh2T1VjQm92OXhpUEg4dWxzSG00SXZLNVNQTStYNGJPMm1vdUdNVWtQcmxXMU1pcVBsNUdYMG9CR1hDODdyNApPUHhtYmRiOFoyTXRqdXQ4ZUFqS094MFZCZ2FqakRlL0hFS2hveG02d1ZFSm9CZXNmQkVIV1FJN25ZbWQyck5LCnBUdXdxdDJ5dnBNa2JST3dITXpOY29PUU1RNlJOQ1hTVlltVmVvZTdBb0dCQUtZamZsWHNmaHFXVnBabzJXRU4KSTVzNEViQlBBbkEyUFZ4dzZWM1RtRDNzUGlubWdrbUhQbzhQQ3ltQy9BNXFpc0dwQWZVNWM4TStIZ013dWtDNgpNMlE4aHQvQVRXdHlDcTFlRnJoMk9iTTlhWE8vRmUxUGlZU2l1eFlMNlQ2TzZjbVA4ZisvMlBadUFZTDd5YWc1CnkrNU5JbGpYVWVMYUI2YUhuZUFYd01XSEFvR0JBTXdQVmVYaSt2KzIzZUtUNUpLM3hWNVVWQStaNFRyMWZwVlIKaDRiOTJESW9VcmpzUzR6bkQwLzNOSWJLN2ZyZlpYbmh1Z0hQWGl3eUhnQlg5V1RSUTZsRzFhLzllVTM0d1RLUwpJZ3lIM3dVVDB3VlMzS1Z5dEgxbmNGVWFEKzBnSDUveFNoQUxZSXNSN2EwT2N3V1E4U1JDblJ1QmVKU212YlkwCjNHMU1iL0pCQW9HQUNkdkRoNVpSRVZzNGlGa25SSDcxQmNyeWFZN1pVbkQ1SzZSUkNHY0krRURQQ3FBOTZvK2MKdnBLSm5pLzVJK01USm8vQ3QrTENXNm56U3ZOS3dzT0N1eVJqSFRnaHZMUU5ESGdZZ2xJQll3a0l3bklFOXUzMQpiQzJIVlg0K3BEVjQzUCtna2t0Uks4ZG1ha2diNUF0dGtVNkJXLzdaL0VsSE4vNW4rTTNhYVpZPQotLS0tLUVORCBSU0EgUFJJVkFURSBLRVktLS0tLQo=
kind: Secret
metadata:
  annotations:
    cert-manager.io/alt-names: localhost,backend.smallstep.com
    cert-manager.io/certificate-name: backend-smallstep-com
    cert-manager.io/common-name: backend.smallstep.com
    cert-manager.io/ip-sans: 127.0.0.1
    cert-manager.io/issuer-kind: CertificateRequest
    cert-manager.io/issuer-name: step-issuer
    cert-manager.io/uri-sans: ""
  creationTimestamp: "2019-08-14T01:02:03Z"
  name: backend-smallstep-com-tls
  namespace: default
  resourceVersion: "430738"
  selfLink: /api/v1/namespaces/default/secrets/backend-smallstep-com-tls
  uid: 751e621c-426c-4493-b253-dc817fd6a64f
type: kubernetes.io/tls
```

### Disabling Approval Check

`StepIssuer` will wait for `CertificateRequest`s to have an [approved condition
set](https://cert-manager.io/docs/concepts/certificaterequest/#approval) before
signing. If using an older version of cert-manager (pre v1.3), you can disable
this check by supplying the command line flag `-disable-approval-check` to the
Issuer Deployment.

### Local development

To run `step-issuer` locally, you can use a [Kind](https://kind.sigs.k8s.io/) cluster. Be sure to create a cluster with at least two workers:

```sh
$ cat <<EOF > kind.yaml
# three node (two workers) cluster config
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
- role: worker
- role: worker
EOF
$ kind create cluster --config kind.yaml
Creating cluster "kind" ...
 ✓ Ensuring node image (kindest/node:v1.25.3) 🖼
 ✓ Preparing nodes 📦 📦 📦
 ✓ Writing configuration 📜
 ✓ Starting control-plane 🕹️
 ✓ Installing CNI 🔌
 ✓ Installing StorageClass 💾
 ✓ Joining worker nodes 🚜
Set kubectl context to "kind-kind"
You can now use your cluster with:

kubectl cluster-info --context kind-kind

Not sure what to do next? 😅  Check out https://kind.sigs.k8s.io/docs/user/quick-start/
```

### Installing from Source

The `make deploy` recipe will install `step-issuer` manually from this repo.

These are the individual steps:

First we install the CRDs:

```sh
kubectl apply -f config/crd/bases/certmanager.step.sm_stepissuers.yaml
kubectl apply -f config/crd/bases/certmanager.step.sm_stepclusterissuers.yaml
```

Then we install the controller:

```sh
kubectl apply -f config/samples/deployment.yaml
```

Or, if you're using [`kustomize`](https://kustomize.io/):

```sh
kustomize build config/default | kubectl apply -f -
```

By default,
the `step-issuer` controller will be installed in the `step-issuer-system` namespace,
but you can edit the YAML files as needed.

```sh
$ kubectl get -n step-issuer-system all
NAME                                                 READY   STATUS    RESTARTS   AGE
pod/step-issuer-controller-manager-9d74f5bff-hnk2c   2/2     Running   0          1m

NAME                                                     TYPE        CLUSTER-IP     EXTERNAL-IP   PORT(S)    AGE
service/step-issuer-controller-manager-metrics-service   ClusterIP   10.96.212.99   <none>        8443/TCP   1m

NAME                                             READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/step-issuer-controller-manager   1/1     1            1           1m

NAME                                                       DESIRED   CURRENT   READY   AGE
replicaset.apps/step-issuer-controller-manager-9d74f5bff   1         1         1       1m
```
