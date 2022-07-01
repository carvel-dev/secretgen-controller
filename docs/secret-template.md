## SecretTemplate

As of v0.9.0+, the `SecretTemplate` is now avaliable under the `secretgen.carvel.dev` API group.

The `SecretTemplate` API allows new Secrets to be composed from data residing in existing Kubernetes Resources, including other Secrets.

Secrets are a common method of encapsulating and inputing sensitive data into other Kubernetes Resource via a reference, or to a process via [volume mounting](https://kubernetes.io/docs/concepts/configuration/secret/#using-secrets-as-files-from-a-pod). However the required information may be contained in a number of other Kubernetes Resources or maybe in an incorrect format.

The CRD `SecretTemplate` provides a way of defining "input resources" (other Kubernetes Resources) and allows the templating out of a new Secret using information found on these resources. It will continuously pick up changes to these resources and update the templated Secret as necessary.

### Example

```yaml
---
apiVersion: v1
kind: Secret
metadata:
  name: password
data:
  password: dG9wU2VjcmV0Cg==
---
apiVersion: v1
kind: Secret
metadata:
  name: username
stringData:
  username: my-user

#! reads two secrets and creates a secret from them
---
apiVersion: secretgen.carvel.dev/v1alpha1
kind: SecretTemplate
metadata:
  name: new-secret
spec:
  #! list of resources to read information off
  inputResources:
  - name: username-secret
    ref:
      apiVersion: v1
      kind: Secret
      name: username
  - name: password-secret
    ref:
      apiVersion: v1
      kind: Secret
      name: password
  #! the template that follows a subset of the Secret API
  template:
    #! data is used for templating in data that *is* base64 encoded, most likely Secrets.
    data:
      password: $(.password-secret.data.password)
      username: $(.username-secret.data.username)
```

Above configuration results in a `new-secret` Secret created within `default` namespace:

```console
Namespace  Name           Kind          Owner    Conds.  Rs  Ri  Age
default    new-secret     Secret        cluster  -       ok  -   1d
```

### SecretTemplate

SecretTemplate CRD allows to template out a Secret from information on other APIs.

`metadata` fields:

- `name`: (required; string) Secret by the same name (in the namespace) will be created.

`spec` fields:

- `serviceAccountName` (required; string) Name of the service account used to read the input resources. If not provided, only Secrets can be read on the `.spec.inputResources`.
- `inputResources` (required; array of objects) Array of named Kubernetes API resources to read information off.  The name of an input resource can dynamically reference previous input resources by a JSONPath expression, signified by an opening "$(" and a closing ")". Input Resources are resolved in the order they are defined.
- `template` (optional; subset of Secret API object) A template of the Secret to be created.  Any string value in the subset can reference information off a resource in `.spec.inputResources` using a JSONPath expression, signified by an opening "$(" and a closing ")".

### Further Example

```yaml
#! reads the resources created by an instance of the bitnami helm chart https://github.com/bitnami/charts/tree/master/bitnami/postgresql/ and creates a binding secret https://github.com/servicebinding/spec
#! example chart installed using the command `helm install my-release bitnami/postgresql`
---
apiVersion: secretgen.carvel.dev/v1alpha1
kind: SecretTemplate
metadata:
  name: helm-postgres
spec:
  #! service account with permissions to get/list/watch pods, services, secrets
  serviceAccountName: helm-reader
  #! list of resources to read off, these resources can be dynamically specified based on the fields of previously stated resources
  inputResources:
  - name: pod
    ref:
      apiVersion: v1
      kind: Pod
      name: my-release-postgresql-0
  - name: service
    ref:
      apiVersion: v1
      kind: Service
      name: my-release-postgresql
  - name: secret
    ref:
      apiVersion: v1
      kind: Secret
      #! the name of an input resource can be determined by the data contained in a previous input resource
      name: $(.pod.spec.containers[?(@.name=="postgresql")].env[?(@.name=="POSTGRES_PASSWORD")].valueFrom.secretKeyRef.name)
  #! the template that follows a subset of the Secret API
  template:
    #! annotation and label metadata properties support templating
    metadata:
      labels:
        key1: $(.pod.metadata.name)
      annotations:
        key2: $(.pod.metadata.name)
    #! the type is immutable for now and can't be updated in subsequent reconciliations
    type: postgresql
    #! stringData is used for templating in data that is not base64 encoded
    stringData:
      port: $(.service.spec.ports[?(@.name=="tcp-postgresql")].port)
      database: postgres
      host: $(.service.spec.clusterIP)
      username: postgres
    #! data is used for templating in data that *is* base64 encoded, most likely Secrets.
    data:
      password: $(.secret.data.postgres-password)

#! example RBAC required to allow SecretTemplate to read data from inputresources
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: helm-reader # service account used by SecretTemplate to read input resources, referred to by SecretTemplate

---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: helm-reader
rules:
- apiGroups:
  - ""
  resources:
  - secrets
  - services
  - pods
  verbs:
  - get
  - list
  - watch

---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: helm-reader
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: helm-reader
subjects:
- kind: ServiceAccount
  name: helm-reader
```

Above configuration results in a `helm-postgres` Secret created within `default` namespace:

```console
Namespace  Name           Kind          Owner    Conds.  Rs  Ri  Age
default    helm-postgres  Secret        cluster  -       ok  -   1d
```
