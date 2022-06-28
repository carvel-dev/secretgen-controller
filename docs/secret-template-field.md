### Secret Template Field

By default generated secrets have predefined set of keys. In a lot of cases `Secret` consumers expect specific set of keys within `data`. To control customization of keys within a secret, you can configure use secret template.

Fields:

- `type` (string; optional) Overrides secret resource type
- `stringData` (map[string]string; optional) Overrides secret data. Values go through variable expansion with each type providing a set of variables that can be used. For example `postgresql-password: $(value)` for password type. See "Secret Template" section for each secret type. Available in v0.3.0+ (earlier available `data` key is removed).

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
    type: Opaque
    stringData:
      postgresql-pass: $(value)
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

See [examples](../examples/passwords.yml) for more.
