## Install

Grab latest copy of YAML from the [Releases page](https://github.com/k14s/secretgen-controller/releases) and use your favorite deployment tool (such as [kapp](https://get-kapp.io) or kubectl) to install it.

Example:

```bash
$ kapp deploy -a sg -f https://github.com/k14s/secretgen-controller/releases/download/v0.2.0/release.yml
or
$ kubectl apply -f https://github.com/k14s/secretgen-controller/releases/download/v0.2.0/release.yml
```

### Advanced

`release.yml` is produced with [ytt](https://get-ytt.io) and [kbld](https://get-kbld.io) at the time of the release. You can use these tools yourself and customize kapp controller configuration if default one does not fit your needs.

Example:

```
$ git clone ...
$ kapp deploy -a sg -f <(ytt -f config/ | kbld -f-)
```

Next: [Walkthrough](walkthrough.md)
