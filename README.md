![logo](docs/CarvelLogo.png)

# secretgen-controller

- Slack: [#carvel in Kubernetes slack](https://slack.kubernetes.io)
- [Docs](docs/README.md) with topics about installation, config, etc.
- Install: see [Install instructions](docs/install.md)
- Status: Supported
- Backlog: [See what we're up to](https://github.com/orgs/carvel-dev/projects/1/views/1?filterQuery=repo%3A%22carvel-dev%2Fsecretgen-controller%22).

`secretgen-controller` provides CRDs to specify what secrets need to be on cluster (generated or not).

Features:

- supports generating certificates, passwords, RSA keys and SSH keys
- supports exporting and importing secrets across namespaces
- exporting/importing registry secrets across namespaces
- supports generating secrets from data residing in other Kubernetes resources

More details in [docs](docs/README.md).

### Join the Community and Make Carvel Better
Carvel is better because of our contributors and maintainers. It is because of you that we can bring great software to the community.
Please join us during our online community meetings. Details can be found on our [Carvel website](https://carvel.dev/community/).

You can chat with us on Kubernetes Slack in the #carvel channel and follow us on Twitter at @carvel_dev.

Check out which organizations are using and contributing to Carvel: [Adopter's list](https://github.com/carvel-dev/carvel/blob/develop/ADOPTERS.md)
