### Certificate

`spec` fields:

- `caRef` (corev1.LocalObjectReference; optional) specifies name of a CA certificate. Used by intermediate CAs or leaf certificates
- `isCA` (bool; optional) specifies whether certificate is a CA. Set to true for root or intermediate CA certificates. If set to `true`, key usage will be set to `x509.KeyUsageCertSign` and `x509.KeyUsageCRLSign`, otherwise key usage is set to `x509.KeyUsageKeyEncipherment` and `x509.KeyUsageDigitalSignature`.
- `commonName` (string; optional) specifies certificate's CN field
- `organization` (string; optional) specifies certificate's Organization field
- `alternativeNames` (array of strings; optional) specifies certificate's alternative names field (IPs or DNS names)
- `extendedKeyUsage` (array of strings; optional) specifies certificate's extended key usage field (`client_auth` and `server_auth` are supported options)
- `duration` (int64; optional) specifies number of days certificate will be valid from now. By default certificate expires in 365 days.
- [`secretTemplate`](secret-template.md)

3072-bit RSA key backs each certificate.

#### Secret Template

Available variables:

- `$(certificate)`
- `$(privateKey)`

#### Examples

Root CA certificate:

```
apiVersion: secretgen.k14s.io/v1alpha1
kind: Certificate
metadata:
  name: root-ca-cert
spec:
  isCA: true
```

Intermediate CA certificate:

```
apiVersion: secretgen.k14s.io/v1alpha1
kind: Certificate
metadata:
  name: inter-ca-cert
spec:
  isCA: true
  caRef:
    name: root-ca-cert
```

Leaf certificate:

```
apiVersion: secretgen.k14s.io/v1alpha1
kind: Certificate
metadata:
  name: inter-ca-cert
spec:
  caRef:
    name: root-ca-cert
  alternativeNames:
  - app1.svc.cluster.local
```

Leaf certificate with custom secret projection:

```
apiVersion: secretgen.k14s.io/v1alpha1
kind: Certificate
metadata:
  name: inter-ca-cert
spec:
  caRef:
    name: root-ca-cert
  alternativeNames:
  - app1.svc.cluster.local
  secretTemplate:
    stringData:
      crt: $(certificate)
      key: $(privateKey)
```
