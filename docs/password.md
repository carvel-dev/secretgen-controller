### Password

Password CRD allows to generate alpha-numeric passwords of particular length.

`spec` fields:

- `length` (optional; default is 40) number of characters

Example:

```yaml
apiVersion: secretgen.k14s.io/v1alpha1
kind: Password
metadata:
  name: long-user-password
spec:
  length: 124
```
