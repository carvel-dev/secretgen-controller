---
name: Release Checklist
about: Checklist for release secretgen-controller
title: ''
labels: carvel, release
assignees: ''

---

## Releasing a new minor / major:
- [ ] OSS Release
    - [ ] [Releasing via workflow](https://github.com/carvel-dev/secretgen-controller/blob/develop/docs/dev.md#release).
    - [ ] Close any GitHub issues that have been delivered.
    - [ ] Add a link to the release on the issue.
- [ ] Update Documentation by [generating a new docs version](https://hackmd.io/uVpvITUuR4Cbwzkzb7MEpQ?view#Generate-new-docs-version)
- [ ] [Communicate in Slack](https://hackmd.io/uVpvITUuR4Cbwzkzb7MEpQ?view#Communicate-in-Slack)
- [ ] [Add to "Announcements" in Next Community Meeting Agenda](https://hackmd.io/uVpvITUuR4Cbwzkzb7MEpQ?view#Announce-in-community-meeting)

## Post Release:
- [ ] Create a Pull Request for [Tanzu Community Edition](https://github.com/vmware-tanzu/community-edition)
    
    #### Versions > 0.10.x:
    - [ ] edit `addons/packages/secretgen-controller/vendir.yml` to include the newly released version.
    - [ ] run `vendir sync`
    - [ ] manually add a `README.MD` to the newly created folder for this version. This `README.MD` can likely be copied from a previous version (add any new / updated schema values to the readme).

    #### Versions < 0.10.x:
    - [ ] Create a new folder, and copy all the contents from the latest previous version e.g `cp -r addons/packages/secretgen-controller/0.10.1 addons/packages/secretgen-controller/0.10.2`.
    - [ ] Update `bundle/vendir.yaml` to the newly released tag version.
    - [ ] Update `addons/packages/secretgen-controller/<your-version>/package.yaml` to the newly released tag version.
    - [ ] Run `make vendir-sync-package PACKAGE=secretgen-controller VERSION=<your-version>`
    - [ ] Run `make lock-package-images PACKAGE=secretgen-controller VERSION=<your-version>`. This will update the contents of `addons/packages/secretgen-controller/0.10.1/bundle/.imgpkg/images.yml`.
    - [ ] Ensure there is only one item in the `images` array for `image.yml` above and that it is the correct sha for the released version. If there are multiple images in the `image.yml` please rerun the previous step.
    - [ ] Update `spec.template.spec.initContainers[0].image` to the release image sha if the value is not the same as above sha.
    - [ ] Run `make push-package PACKAGE=secretgen-controller VERSION=<your-version> TAG=<your-version>`.
        - > NOTE: Ensure you are logged into the registry `docker login projects.registry.vmware.com`. Contact the [#tanzu-community-edition](https://kubernetes.slack.com/archives/C02GY94A8KT) slack on the kubernetes workspace if you do not have access to push.
    - [ ] Once you push the image, copy the above SHA generated and replace the image sha in `addons/packages/secretgen-controller/<your-version>/package.yaml`.
    - [ ] Verify the generated package looks correct by running `ytt --ignore-unknown-comments -f addons/packages/secretgen-controller/<your-version>/bundle/config > test.yaml`. (Don't include this test.yaml file in the PR)

## Releasing a patch version and backporting a CVE:
- [ ] Validate which branch lines to backport the CVE to. Based on our [private confluence page](https://confluence.eng.vmware.com/x/FyIuSQ).
- [ ] For each line, e.g `v0.9.x`, `v0.10.x`, do the following:
    - [ ] Validate that the branch contains the latest patches, that no newer code was forgotten to be merged back in.
    - [ ] `git checkout v0.9.x`.
    - [ ] `git checkout -b v0.9.<next-patch-version>`.
    - [ ] Make the necessary fixes / cherry-picks.
    - [ ] `git push origin v0.9.<next-patch-version>`.
    - [ ] Make a PR.
    - [ ] Once approved, merge the changes back to the `v0.9.x` branch and `git push` the branch and delete your temporary branch used in the PR.
    - [ ] To Release: follow the instructions FROM THE BRANCH YOU ARE UPDATING at `docs/dev.md#release` in the repository. These will contain the relevant steps at each point of time in the project's history, e.g when updating `v0.9.x` the url will look like: https://github.com/carvel-dev/secretgen-controller/blob/v0.9.x/docs/dev.md#release
