### Password

Password CRD allows to generate alpha-numeric passwords of particular length.

`spec` fields:

- `length` (optional; default is 40) number of characters
- [`secretTemplate`](secret-template.md)

#### Secret Template

Data value references:

- `value`

#### Examples

```yaml
apiVersion: secretgen.k14s.io/v1alpha1
kind: Password
metadata:
  name: long-user-password
spec:
  length: 124
```

With custom secret projection:

```yaml
apiVersion: secretgen.k14s.io/v1alpha1
kind: Password
metadata:
  name: long-user-password
spec:
  length: 124
  secretTemplate:
    data:
      postgresql-password: value
```
