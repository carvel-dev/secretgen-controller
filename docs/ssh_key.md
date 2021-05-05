### SSH Key

Please let us know in issues what kind of configurability is wanted.

`spec` fields:

- [`secretTemplate`](secret-template.md)

#### Secret Template

Available variables:

- `$(privateKey)`
- `$(authorizedKey)`

#### Example

```
apiVersion: secretgen.k14s.io/v1alpha1
kind: SSHKey
metadata:
  name: ssh-key
spec: {}
```
