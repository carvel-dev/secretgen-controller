## SecretTemplate

As of v0.9.0+, the `SecretTemplate` is now avaliable under the `secretgen.carvel.dev` API group.

Secrets are a common method of encapsulating and inputing sensitive data into an API via reference, or to a process via volume mounting.  In order to make compatible Secrets for these use cases, it often requires imperatively reading of other resources (including other secrets) and imperatively creating the compatible secret using these resources' information.  `SecretTemplate` aims to provide a declarative way of doing this process.

The CRD `SecretTemplate` provides a way of provides "input resources" (other resources on the API) and templating out a Secret using information found on these resources.  It will then pick up changes to these resources and update the templatde Secret as necessary.

### Example

```yaml
---
apiVersion: v1
kind: Secret
metadata:
  name: password-secret
data:
  password: dG9wU2VjcmV0Cg==
---
apiVersion: v1
kind: Secret
metadata:
  name: username-secret
stringData:
  username: my-user

#! reads two secrets and creates a secret from them
---
apiVersion: secretgen.carvel.dev/v1alpha1
kind: SecretTemplate
metadata:
  name: pod-input
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
    #! data is used for templating in data that *is* base64 encoded, most like Secrets.
    data:
      password: $(.password-secret.data.password)
      username: $(.username-secret.data.username)
```

Above configuration results in a `helm-postgres` Secret created within `default` namespace:

```console
Namespace  Name           Kind          Owner    Conds.  Rs  Ri  Age
default    helm-postgres  Secret        cluster  -       ok  -   1d
```

### SecretTemplate

SecretTemplate CRD allows to template out a Secret from information on other APIs.

`metadata` fields:

- `name`: (required; string) Secret by the same name (in the namespace) will be created.

`spec` fields:

- `serviceAccountName` (required; string) Name of the service account used to read the input resources. If not provided, only Secrets can be read on the `.spec.inputResources`.
- `inputResources` (required; array of objects) Array of named Kubernetes API resources to read information off.  The name of an input resource can dynamically reference previous input resources by a JSONPath expression, signified by an opening "$(" and a closing ")".
- `template` (optional; subset of Secret API object) A template of the Secret to be created.  Any string value in the subset can reference information off a resource in `.spec.inputResources` using a JSONPath expression, signified by an opening "$(" and a closing ")".

### Further Example

```yaml
#! reads the resources created by an instance of the bitnami helm chart https://github.com/bitnami/charts/tree/master/bitnami/postgresql/ and creates a binding secret https://github.com/servicebinding/spec
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
      name: postgres-postgresql-0
  - name: service
    ref:
      apiVersion: v1
      kind: Service
      name: postgres-postgresql
  - name: secret
    ref:
      apiVersion: v1
      kind: Secret
      name: $(.pod.spec.containers[?(@.name=="postgresql")].env[?(@.name=="POSTGRES_PASSWORD")].valueFrom.secretKeyRef.name)
  #! the template that follows a subset of the Secret API
  template:
    #! the type is immutable for now and can't be updated in subsequent reconciliations
    type: postgresql
    #! stringData is used for templating in data that is not base64 encoded
    stringData:
      port: $(.service.spec.ports[0].port)
      database: $(.pod.spec.containers[0].env[?(@.name=="POSTGRES_DB")].value)
      host: $(.service.spec.clusterIP)
      username: $(.pod.spec.containers[0].env[?(@.name=="POSTGRES_USER")].value)
    #! data is used for templating in data that *is* base64 encoded, most like Secrets.
    data:
      password: $(.secret.data.password)
```

Above configuration results in a `helm-postgres` Secret created within `default` namespace:

```console
Namespace  Name           Kind          Owner    Conds.  Rs  Ri  Age
default    helm-postgres  Secret        cluster  -       ok  -   1d
```
