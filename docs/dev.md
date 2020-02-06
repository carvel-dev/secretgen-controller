## Development & Deploy

Install ytt, kbld, kapp beforehand (https://k14s.io).

```
./hack/build.sh # to build locally

# add `-v image_repo=docker.io/username/secretgen-controller` with your registry to ytt invocation inside
./hack/deploy.sh # to deploy

export SECRETGEN_E2E_NAMESPACE=secretgen-test
./hack/test-all.sh
```

### TODO

- regenerate if params changed
- certificate rotation?
- secret name is static -> does not trigger change
- kapp versioned certificate
