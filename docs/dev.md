## Development & Deploy

Install ytt, kbld, kapp beforehand (https://k14s.io).

```
./hack/build.sh # to build locally

# add `-v image_repo=docker.io/username/secretgen-controller` with your registry to ytt invocation inside
./hack/deploy.sh # to deploy

export SECRETGEN_E2E_NAMESPACE=secretgen-test
./hack/test-all.sh
```

## Release
Tag the release - it's necessary to do this first because the release process
uses the tag to record the version.
`git tag "v1.2.3"`

Log in to the k8slt dockerhub account (vmware employees should have the password
via lastpass):
`docker login -u "k8slt" docker.io`

Run the release script:  `hack/build-release.sh`

Copy the resulting yaml into the repo:
cp tmp/release.yml releases/1.2.3.yml

Test the release artifact to ensure it actually works
`kapp deploy -a secretgen -f releases/1.2.3.yml`
Ensure the deployment and build worked by running the e2e tests:
you may need to `export SECRETGEN_E2E_NAMESPACE=secretgen-test`
then `./hack/test-e2e.sh`.

After verifying, commit: `git add releases; git commit -m "release 1.2.3"`
and push: `git push origin HEAD --tags`

#### Alpha Releases
Similar to above but we have a separate `build-alpha-release.sh` script and
`alpha-releases` folder. Version names should be like `v1.2.3-alpha.1`


### TODO

- regenerate if params changed
- certificate rotation?
- secret name is static -> does not trigger change
- kapp versioned certificate
