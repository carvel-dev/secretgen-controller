### Secret Template

By default generated secrets have predefined set of keys. In a lot of cases `Secret` consumers expect specific set of keys within `data`. To control customization of keys within a secret, you can configure use secret template.

Fields:

- `type` (string; optional) Overrides secret resource type
- `data` (map[string]string; optional) Overrides secret data. Keys are actual secret keys, and values are reference values that indicate what value should be inserted. For example `postgresql-password: value` in a password filed. See "Secret Template" section for each secret type.

#### Example

```yaml
apiVersion: secretgen.k14s.io/v1alpha1
kind: Password
metadata:
  name: pg-password
spec: {}
```

would by default produce:

```
apiVersion: v1
kind: Secret
metadata:
  name: pg-password
data:
  password: xxx...
```

With custom secret projection:

```yaml
apiVersion: secretgen.k14s.io/v1alpha1
kind: Password
metadata:
  name: pg-password
spec:
  secretTemplate:
    data:
      postgresql-pass: value
```

would produce:

```
apiVersion: v1
kind: Secret
metadata:
  name: pg-password
data:
  postgresql-pass: xxx...
```
