# step-issuer

Step Issuer is a [cert-manager's](https://github.com/jetstack/cert-manager)
CertificateRequest controller that uses [step
certificates](https://github.com/smallstep/certificates) (a.k.a. `step-ca`) to
sign the certificate requests.

## Getting started

In this guide, we assume that you have a [Kubernetes](https://kubernetes.io/)
environment with a [cert-manager](https://github.com/jetstack/cert-manager)
version supporting CertificateRequest issuers, cert-manager v0.9.0 or higher.

### Installing step certificates

Step Issues uses [step certificates](https://github.com/smallstep/certificates)
as the Certificate Authority or CA in charge of signing the CertificateRequest
resources. To install `step certificates` the easiest way is to use helm:

```sh
helm repo add smallstep  https://smallstep.github.io/helm-charts
helm repo update
helm install --name step-certificates smallstep/step-certificates
```

Please refer to [step certificates](https://github.com/smallstep/certificates)
for other installation methods, and more advanced features.

With `step certificates` installed, we need to get the CA URL, the root
certificate, and a provisioner name, kid and password. The default Helm
installation will only configure one provisioner named `admin`. It is
recommended to add a separate provisioner for cert-manager, but for this guide
we will use the default one.

With the previous `helm install` the CA URL will be
<https://step-certificates.default.svc.cluster.local>.

The root certificate can be obtained from the `step-certificates-certs`
ConfigMap or running:

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

To configure the step issuer we will use the base64 version of the root certificate:

```sh
$ kubectl get -o jsonpath="{.data['root_ca\.crt']}" configmaps/step-certificates-certs | base64
LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJpekNDQVRHZ0F3SUJBZ0lRTytFQWg4eS8wVjlQMFhwSHJWajVOVEFLQmdncWhrak9QUVFEQWpBa01TSXcKSUFZRFZRUURFeGxUZEdWd0lFTmxjblJwWm1sallYUmxjeUJTYjI5MElFTkJNQjRYRFRFNU1EZ3hNekU1TVRVdwpNbG9YRFRJNU1EZ3hNREU1TVRVd01sb3dKREVpTUNBR0ExVUVBeE1aVTNSbGNDQkRaWEowYVdacFkyRjBaWE1nClVtOXZkQ0JEUVRCWk1CTUdCeXFHU000OUFnRUdDQ3FHU000OUF3RUhBMElBQkFNVkw3VzBQbTNvSlVmSTR3WGQKa2xERW5uNVhTbWo4NlgwYW1DQTBnY08xdElUUG1DVzNCcGU0cE9vV1V2WlZlUWRvU2NxN3pua1V0Mi9HMnQxTgo3MWlqUlRCRE1BNEdBMVVkRHdFQi93UUVBd0lCQmpBU0JnTlZIUk1CQWY4RUNEQUdBUUgvQWdFQk1CMEdBMVVkCkRnUVdCQlJ1Y1ByVm5QdlpOMHI0QVU5TGcyL2VCcng3a2pBS0JnZ3Foa2pPUFFRREFnTklBREJGQWlCUlJBdGsKNXpMY0doQ2FobVBuVzIwZExpdEMzRVdNaVE0bERwN2FFeitFUEFJaEFJOWZWczVxb0l0bVQ4anA2WktVNVEydQphRFBrOGsyQ25OMjdyRnNZV3VwTAotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==
```

The provisioner information can be obtained from the `step-certificates-config`
ConfigMap or running:

```sh
$ kubectl get -o jsonpath="{.data['ca\.json']}" configmaps/step-certificates-config | jq .authority.provisioners
[
  {
    "type": "jwk",
    "name": "admin",
    "key": {
      "use": "sig",
      "kty": "EC",
      "kid": "N6I99Yuk7iGDMk_eW3QaN2admCsrC9UuDN27dlFXUOs",
      "crv": "P-256",
      "alg": "ES256",
      "x": "5QWf_SOxbYdbnrqHZOlhq3QEJ6D5iNJRhziDwDAUbmU",
      "y": "DUkNegtRu-A37PRlDf4B_4sHSaFairojgxtZjih1I0Y"
    },
    "encryptedKey": "eyJhbGciOiJQQkVTMi1IUzI1NitBMTI4S1ciLCJjdHkiOiJqd2sranNvbiIsImVuYyI6IkEyNTZHQ00iLCJwMmMiOjEwMDAwMCwicDJzIjoiOC1sVmgtdE9ub3J5ZUJQeElsQlRFZyJ9.1j4QivzNn7cGxzAXg_qUylMFhCVKoci5g1AKXkNPSl9G_yf3Jz0oYQ.tkN0fhymCqYBfBRC.mMfe_mz6S58LboMQbwOkOt4rLq0M5xkViWR6gx5f6b0No9Rz2tXm7VeXs8qS7oYmri8Hw5wjykY6H0kgCS__taIkwhLHxXVQiwz_3ivu5Jqam2xSu4Z_vtajfLaT45yUhRpglQm7qcfDQhVgN_klCDeis4qhGTflrSnQIuE_wq-QnA91Rh8Pmu_Ky-YnJww3WBitUdufkEHAQFGZl532U0AvsNQKLxDOXOVt-D8RTIjTdjUX4lUPM_FIFHVj6hMsatpe4FSQYGjIFZqTrqz8EOq8s34nAx4G12xY7ciG906zza7C07fnKKYcvhUlFLXCGWaJlKg4ezdK9nScqLY.kloMyiDr-wVK3LRXPNzhzg"
  }
]
```

And finally we need the password to decrypt the provisioner private key, this is
available in the secret `step-certificates-provisioner-password`, and can be
obtained running:

```sh
$ kubectl get -o jsonpath='{.data.password}' secret/step-certificates-provisioner-password | base64 --decode
MfKmjQrR1iw3ZvTd4CImQfhwIbdq2FRp
```

We won't use the plain password to configure the step-issuer, we will be
referencing the same secret.

To recap, we got:

* The CA url <https://step-certificates.default.svc.cluster.local>
* The root certificate in base64
* The provisioner name `admin`
* The provisioner kid `N6I99Yuk7iGDMk_eW3QaN2admCsrC9UuDN27dlFXUOs`
* And the provisioner password secret `step-certificates-provisioner-password`
  and key `password`

### Installing step issuer

Finally, we need to install the step issuer. The easiest way to install it is
running `make deploy`, but we're going to run the individual steps here:

First we install the CRDs:

```sh
kubectl apply -f config/crd/bases
```

Then we install the controller:

```sh
kubectl apply -f config/samples/deployment.yaml
# or with kustomize
# kustomize build config/default | kubectl apply -f -
```

By default, the step-issuer controller will be installed in the namespace
`step-issuer-system`, but you can edit the YAML files to your convenience.

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

### Adding a StepIssuer

Now, we're going to use all the configuration values that we got after
installing `step certificates` and use them to configure our StepIssuer. With
the previous values the YAML will look like:

```yaml
apiVersion: certmanager.step.sm/v1beta1
kind: StepIssuer
metadata:
  name: step-issuer
  namespace: step-issuer-system
spec:
  # The CA URL.
  url: https://step-certificates.default.svc.cluster.local
  # The base64 encoded version of the CA root certificate in PEM format.
  caBundle:  LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJpekNDQVRHZ0F3SUJBZ0lRTytFQWg4eS8wVjlQMFhwSHJWajVOVEFLQmdncWhrak9QUVFEQWpBa01TSXcKSUFZRFZRUURFeGxUZEdWd0lFTmxjblJwWm1sallYUmxjeUJTYjI5MElFTkJNQjRYRFRFNU1EZ3hNekU1TVRVdwpNbG9YRFRJNU1EZ3hNREU1TVRVd01sb3dKREVpTUNBR0ExVUVBeE1aVTNSbGNDQkRaWEowYVdacFkyRjBaWE1nClVtOXZkQ0JEUVRCWk1CTUdCeXFHU000OUFnRUdDQ3FHU000OUF3RUhBMElBQkFNVkw3VzBQbTNvSlVmSTR3WGQKa2xERW5uNVhTbWo4NlgwYW1DQTBnY08xdElUUG1DVzNCcGU0cE9vV1V2WlZlUWRvU2NxN3pua1V0Mi9HMnQxTgo3MWlqUlRCRE1BNEdBMVVkRHdFQi93UUVBd0lCQmpBU0JnTlZIUk1CQWY4RUNEQUdBUUgvQWdFQk1CMEdBMVVkCkRnUVdCQlJ1Y1ByVm5QdlpOMHI0QVU5TGcyL2VCcng3a2pBS0JnZ3Foa2pPUFFRREFnTklBREJGQWlCUlJBdGsKNXpMY0doQ2FobVBuVzIwZExpdEMzRVdNaVE0bERwN2FFeitFUEFJaEFJOWZWczVxb0l0bVQ4anA2WktVNVEydQphRFBrOGsyQ25OMjdyRnNZV3VwTAotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==
  # The provisioner name, kid, and a reference to the provisioner password secret.
  provisioner:
    name: admin
    kid: N6I99Yuk7iGDMk_eW3QaN2admCsrC9UuDN27dlFXUOs
    passwordRef:
      name: step-certificates-provisioner-password
      key: password
```

Note that your configuration will be different, but let's apply ours:

```sh
$ kubectl apply -f config/samples/stepissuer.yaml
stepissuer.certmanager.step.sm/step-issuer created
```

Moments later you should be able to see the `status` property in the resource:

```sh
kubectl get stepissuers.certmanager.step.sm step-issuer -o yaml
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

At this time Step Issuer is ready to sign certificates.

### Creating our first certificate

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
$ cat internal.csr | base64
LS0tLS1CRUdJTiBDRVJUSUZJQ0FURSBSRVFVRVNULS0tLS0KTUlJQkVEQ0J0d0lCQURBaE1SOHdIUVlEVlFRREV4WnBiblJsY201aGJDNXpiV0ZzYkhOMFpYQXVZMjl0TUZrdwpFd1lIS29aSXpqMENBUVlJS29aSXpqMERBUWNEUWdBRVdZYU9lcGhGZmh2ZlN5djdob1BPcEtBOEl3U0JCZlRWCnhMVzNST1lHUDFNNUR1RkU4TkZTWUlDRTJIdzd4ZFA5b2FTeSt2NURvdTVLWk5yNTNEMi80S0EwTURJR0NTcUcKU0liM0RRRUpEakVsTUNNd0lRWURWUjBSQkJvd0dJSVdhVzUwWlhKdVlXd3VjMjFoYkd4emRHVndMbU52YlRBSwpCZ2dxaGtqT1BRUURBZ05JQURCRkFpQXFTRHJKMjltSzVRTTJXRUw1bXRXVnQ5Rlp0cEJXYVBXVVdRTnV2SEpsClpBSWhBUDk1T1BHa0NabkRpTHh5ZHdQaWVjdHVlK2M0SHBVd2RhYU40Sm1FMWZ5aAotLS0tLUVORCBDRVJUSUZJQ0FURSBSRVFVRVNULS0tLS0K
```

And put everything together:

```yaml
apiVersion: certmanager.k8s.io/v1alpha1
kind: CertificateRequest
metadata:
  name: internal-smallstep-com
  namespace: default
spec:
  # The base64 encoded version of the certificate request in PEM format.
  csr: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURSBSRVFVRVNULS0tLS0KTUlJQkVEQ0J0d0lCQURBaE1SOHdIUVlEVlFRREV4WnBiblJsY201aGJDNXpiV0ZzYkhOMFpYQXVZMjl0TUZrdwpFd1lIS29aSXpqMENBUVlJS29aSXpqMERBUWNEUWdBRVdZYU9lcGhGZmh2ZlN5djdob1BPcEtBOEl3U0JCZlRWCnhMVzNST1lHUDFNNUR1RkU4TkZTWUlDRTJIdzd4ZFA5b2FTeSt2NURvdTVLWk5yNTNEMi80S0EwTURJR0NTcUcKU0liM0RRRUpEakVsTUNNd0lRWURWUjBSQkJvd0dJSVdhVzUwWlhKdVlXd3VjMjFoYkd4emRHVndMbU52YlRBSwpCZ2dxaGtqT1BRUURBZ05JQURCRkFpQXFTRHJKMjltSzVRTTJXRUw1bXRXVnQ5Rlp0cEJXYVBXVVdRTnV2SEpsClpBSWhBUDk1T1BHa0NabkRpTHh5ZHdQaWVjdHVlK2M0SHBVd2RhYU40Sm1FMWZ5aAotLS0tLUVORCBDRVJUSUZJQ0FURSBSRVFVRVNULS0tLS0K
  # The duration of the certificate
  duration: 24h
  # If the certificate will be a CA or not.
  # Step certificates won't accept a certificate request if this value is true,
  # you can also omit this.
  isCA: false
  # A reference to the issuer in charge of signing the CSR.
  issuerRef:
    group: certmanager.step.sm
    name: step-issuer
```

We apply it using kubectl:

```sh
$ kubectl apply -f config/samples/certificaterequest.yaml
certificaterequest.certmanager.k8s.io/internal-smallstep-com configured
```

And moments later the bundled signed certificate with the intermediate as well
as the root certificate will be available in the resource:

```sh
$ kubectl get certificaterequests.certmanager.k8s.io internal-smallstep-com -o yaml
apiVersion: certmanager.k8s.io/v1alpha1
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

**Now you are ready to use the TLS certificate in your app.**
