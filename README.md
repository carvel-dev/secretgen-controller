# secretgen-controller

- Slack: [#carvel in Kubernetes slack](https://slack.kubernetes.io)
- [Docs](docs/README.md) with topics about installation, config, etc.
- Install: see [Install instructions](docs/install.md)
- Status: Experimental

`secretgen-controller` provides CRDs to specify what secrets need to be on cluster (generated or not).

Features:

- supports generating certificates, passwords, RSA keys and SSH keys
- supports exporting and importing secrets across namespaces

More details in [docs](docs/README.md).
