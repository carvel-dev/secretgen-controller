## Development & Deploy

Install ytt, kbld, kapp beforehand (https://k14s.io).

```
./hack/build.sh # to build locally

# add `-v image_repo=docker.io/username/secretgen-controller` with your registry to ytt invocation inside
./hack/deploy.sh # to deploy

./hack/dev-deploy.sh # to deploy quickly for iterating

export SECRETGEN_E2E_NAMESPACE=secretgen-test
./hack/test-all.sh
```

Note there's two deploy scripts above. You may wish to use the (slower)
`deploy.sh` at the beginning and end of iterating on a feature, as it will run
gofmt, check your mods, and do a clean / hermetic build. However once you're in
the loop of iterating and re-deploying small changes, you will definitely prefer
the `dev-deploy.sh` script as it is much faster.

## Release
Ensure `git status` shows a "clean" / no untracked changes status.

Tag the release - it's necessary to do this first because the release process
uses the tag to record the version.
`git tag "v1.2.3"`

Create a new
[PAT](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token)
in the github UI, ensuring that it has "package:write"
permissions. After you click "package:write" the other necessary permissions
will auto-select. You can set a 7-day expiration, but you should delete the
token after the release for maximum security.

`export CR_PAT=<your-token-here>`
and then run a docker login:
`echo $CR_PAT | docker login ghcr.io -u <your-username-here>  --password-stdin`

Ensure that you've linked your command-line to your kubernetes environment:
`eval $(minikube docker-env)` (or similar if you're using kind, etc.)

Run the release script:  `hack/build-release.sh`

Push the tag `git push --tags origin HEAD` and then in the github UI make a new
release from that tag.
In the release, add a triple-tick section at the bottom with the checksum that
was output by the release script and the filename (release.yml, package.yml and metadata.yml,  omit the
`./tmp` prefix). Ensure there's two spaces between the checksum and the filename
to make it interoperable with checksum tooling verification.

In your OS UI, open the secretgen-controller repo tmp folder and drag the
release.yml, package.yml and metadata.yml assets into the github UI release as attached files.

Test the release artifact to ensure it actually works
`kapp deploy -a secretgen -f releases/1.2.3.yml`
Ensure the deployment and build worked by running the e2e tests:
you may need to `export SECRETGEN_E2E_NAMESPACE=secretgen-test`
then `./hack/test-e2e.sh`.

Test the package artifact to ensure it actually works
`kapp delete -a secretgen`

Create a PackageInstall file (pkgi.yml) which will be used to install of Secretgen-controller package

```
---
apiVersion: packaging.carvel.dev/v1alpha1
kind: PackageInstall
metadata:
  name: secretgen-controller
spec:
  serviceAccountName: cluster-admin-sa
  packageRef:
    refName: secretgen-controller.carvel.dev
    apiVersion: packaging.carvel.dev/v1alpha1
    versionSelection:
      constraints: ">0.8.0"
```

> Note: cluster-admin-sa comes from deploying https://github.com/vmware-tanzu/carvel-kapp-controller/tree/develop/examples/rbac

`kapp deploy -a secretgen-package -f package.yml -f metadata.yml -f pkgi.yml`
Ensure the deployment and build worked by running the e2e tests:
you may need to `export SECRETGEN_E2E_NAMESPACE=secretgen-test`
then `./hack/test-e2e.sh`.

After verifying, commit: `git add releases; git commit -m "release 1.2.3"`
and push: `git push origin HEAD --tags`

Go into the github UI and delete your PAT.

#### Alpha Releases
Similar to above but we have a separate `build-alpha-release.sh` script and
`alpha-releases` folder, and we don't make a github release.
Version names should be like `v1.2.3-alpha.1`

Copy the yaml from the deploy script into the repo:
cp tmp/release.yml alpha-releases/1.2.3.yml

### TODO

- regenerate if params changed
- certificate rotation?
- secret name is static -> does not trigger change
- kapp versioned certificate
