# Operator Install Configuration

## Write the Nomos Resource
The Nomos resource is a Kubernetes Custom Resource Definition (CRD) that defines a Nomos installation. The `spec` field of a Nomos resources specifies the installation parameters for Nomos.

An example config using a github repo with ssh access is below:
```yaml
apiVersion: addons.sigs.k8s.io/v1alpha1
kind: Nomos
metadata:
  name: nomos
  namespace: nomos-system
spec:
  git:
    syncRepo: git@github.com:frankfarzan/foo-corp-example.git
    syncBranch: master
    syncWait: 1
    secretType: ssh
    policyDir: foo-corp
```

`spec` contains a top level field `git`, which is an object with the following properties:

Key                  | Description
-------------------- | -----------
`syncRepo`             | The URL of the Git repository to use as the source of truth. Required.
`syncBranch`           | The branch of the repository to sync from. Default: master.
`policyDir`           | The path within the repository to the top of the policy hierarchy to sync. Default: the root directory of the repository.
`syncWait`           | Period in seconds between consecutive syncs.  Default: 15.
`syncRev`           | Git revision (tag or hash) to check out. Default HEAD.
`secretType`           | The type of secret configured for access to the Git repository. One of "ssh" or "cookiefile". Required.


## Create the Secret File

### Using SSH Secret Type
First, create a nomos-specific private/public key pair.
```console
$ ssh-keygen -t rsa -b 4096 -C "alice@example.com" -N '' -f $HOME/.ssh/id_rsa.nomos
```
Whether to use a different key per cluster is up to the needs of your system.

Next, configure the server where your repository is hosted to recognize the newly created public key, `id_rsa.nomos.pub`. This process depends on how your repository is hosted. For an example: if your repository is hosted on Github, follow [this process](https://help.github.com/articles/adding-a-new-ssh-key-to-your-github-account/) to add the public key to your Github account.

### Using GitCookies Secret Type
The process for acquiring a gitcookie depends on the configuration of your git server your repository is on, but is commonly used as an authentication mechanism for some hosting services, such as Google Cloud Source Repositories and Gerrit. Git Cookies are usually stored in `~/.gitcookies` on the local machine.
