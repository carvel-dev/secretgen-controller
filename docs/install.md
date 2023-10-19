## Install

Grab latest copy of YAML from the [Releases page](https://github.com/vmware-tanzu/carvel-secretgen-controller/releases) and use your favorite deployment tool (such as [kapp](https://get-kapp.io) or kubectl) to install it.

Example:

```bash
$ kapp deploy -a sg -f https://github.com/vmware-tanzu/carvel-secretgen-controller/releases/latest/download/release.yml
or
$ kubectl apply -f https://github.com/vmware-tanzu/carvel-secretgen-controller/releases/latest/download/release.yml
```

### Advanced

`release.yml` is produced with [ytt](https://get-ytt.io) and [kbld](https://get-kbld.io) at the time of the release. You can use these tools yourself and customize secretgen-controller configuration if default one does not fit your needs.

Example:

```bash
$ git clone ...
$ kapp deploy -a sg -f <(ytt -f config/ | kbld -f-)
```

Next: [Walkthrough](walkthrough.md)
