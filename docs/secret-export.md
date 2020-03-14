## SecretExport and SecretRequest

Access to Secret is commonly scoped to its containing namespace (e.g. Pod can only reference Secrets within the same namespace). This constraint is important for maintaing "namespace" boundary.

In some cases an owner of a Secret may want to export it to other namespaces for consumption by other users/programs in the system. Currently there is no builtin way to do this in Kubernetes.

This project introduces two CRDs `SecretExport` and `SecretRequest` that enable sharing of Secrets across namespaces. SecretExport lets system know that particular Secret is "offered" to be shared with one or more specific namespaces. In the destination namespaces, SecretRequest resource lets system know that Secret is allowed to be "copied" into this namespace. By providing a way to express intent of exporting and approving an export, we are able to securely share Secrets between namespaces, without worrying about Secrets getting "stolen" from a different namespace.

Example:

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: user1
---
apiVersion: v1
kind: Namespace
metadata:
  name: user2

#! generate user-password secret upon creation
---
apiVersion: secretgen.k14s.io/v1alpha1
kind: Password
metadata:
  name: user-password
  namespace: user1

#! offer user-password to user2 namespace
---
apiVersion: secretgen.k14s.io/v1alpha1
kind: SecretExport
metadata:
  name: user-password
  namespace: user1
spec:
  toNamespace: user2

#! allow user-password to be created in user2 namespace
---
apiVersion: secretgen.k14s.io/v1alpha1
kind: SecretRequest
metadata:
  name: user-password
  namespace: user2
spec:
  fromNamespace: user1
```

Above configuration results in a `user-password` Secret created within `user2` namespace:

```
Namespace  Name           Kind           Owner    Conds.  Rs  Ri  Age
(cluster)  user1          Namespace      kapp     -       ok  -   1d
^          user2          Namespace      kapp     -       ok  -   1d
user1      user-password  Password       kapp     1/1 t   ok  -   1d
^          user-password  Secret         cluster  -       ok  -   1d
^          user-password  SecretExport   kapp     1/1 t   ok  -   1d
user2      user-password  Secret         cluster  -       ok  -   1d
^          user-password  SecretRequest  kapp     1/1 t   ok  -   1d
```

### SecretExport

SecretExport CRD allows to "offer" secrets for export.

`metadata` fields:

- `name`: (required; string) Secret by the same name (in the namespace) will be offered.

`spec` fields:

- `toNamespace` (optional; string) Destination namespace for offer.
- `toNamespaces` (optional; array of strings) List of destination namespaces for offer.

### SecretRequest

SecretRequest CRD allows to "accept" secrets being exported.

`metadata` fields:

- `name`: (required; string) Secret by the same name (in the namespace) will be imported; must match SecretExport's name.

`spec` fields:

- `fromNamespace` (optional; string) Source namespace; must be one of ServiceExport's destination namespaces.
