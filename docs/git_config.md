# Git Configuration

A sample YAML file for configuring git-based GKE Policy Management is provided
below:

```yaml
contexts:
  - kubeconfig_context_of_your_cluster
git:
  GIT_SYNC_BRANCH: master
  GIT_SYNC_REPO: git@github.com:frankfarzan/foo-corp-example.git
  GIT_SYNC_SSH: true
  KNOWN_HOSTS_FILENAME: $HOME/.ssh/known_hosts
  PRIVATE_KEY_FILENAME: $HOME/.ssh/id_rsa.nomos
  GIT_SYNC_WAIT: 60
  POLICY_DIR: foo-corp
user: youruser@foo-corp.com
```

Note:

*   `contexts` field is a list of clusters where GKE Policy Management will be
    installed. Run `kubectl config get-contexts` to see what contexts are
    available to you.
*   Set `user` field to be set to your username that is valid for authenticating
    to the clusters. This username must be valid on all clusters included in the
    contexts field.
*   You may use `$HOME` to refer to your home directory in the config file.

These are all the supported keys for the the *git* object of the installer
config file.

Key                  | Description
-------------------- | -----------
GIT_SYNC_REPO        | Address of the git repo to sync from in https, ssh, or git format
GIT_SYNC_BRANCH      | Git branch to check out. Default: "master"
GIT_SYNC_WAIT        | Number of seconds between syncs. Default: 15 seconds
GIT_SYNC_SSH         | true if ssh auth should be used to access the repo. Default: true
PRIVATE_KEY_FILENAME | Absolute path to the ssh private key to use for ssh access to the git repo
KNOWN_HOSTS_FILENAME | Absolute path to the ssh known hosts file to use during ssh access to the git repo. If omitted, strict known hosts checking is disabled
GIT_COOKIE_FILENAME  | Absolute path to a [HTTP git cookie file](https://git-scm.com/docs/git-config/2.1.0#git-config-httpcookiefile) to use to authenticate to the git repo. Use only with HTTP auth
POLICY_DIR           | Relative path of root policy directory in the repo

## Config Reference

This section enumerates ConfigMaps and Secrets used by GKE Policy Management.
When using installer, these are automatically created in `nomos-system`
namespace.

### configmap/git-policy-importer

Used by gitpolicyimporter deployment:

Key                        | Description                                                                          | Container
-------------------------- | ------------------------------------------------------------------------------------ | ---------
GIT_SYNC_REPO              | Git repository to clone                                                              | git-sync
GIT_SYNC_BRANCH            | Git branch to check out                                                              | git-sync
GIT_SYNC_REV               | Git revision (tag or hash) to checkout                                               | git-sync
GIT_SYNC_DEPTH             | Use a shallow clone with a history truncated to the specified number of commits      | git-sync
GIT_SYNC_WAIT              | Number of seconds between syncs                                                      | git-sync
GIT_SYNC_MAX_SYNC_FAILURES | Number of consecutive failures allowed before aborting (the first pull must succeed) | git-sync
GIT_SYNC_USERNAME          | Username to use                                                                      | git-sync
GIT_SYNC_PASSWORD          | Password to use                                                                      | git-sync
GIT_SYNC_SSH               | Use SSH for Git operations                                                           | git-sync
GIT_KNOWN_HOSTS            | Enable SSH known_hosts verification                                                  | git-sync
GIT_COOKIE_FILE            | Enable HTTP cookie usage for git access                                              | git-sync
POLICY_DIR                 | Relative path of root policy directory in the repo                                   | policy-importer

### secret/git-creds

Used by gitpolicyimporter deployment:

Key         | Description          | Container
----------- | -------------------- | ---------
ssh         | SSH private key      | git-sync
known_hosts | SSH known hosts file | git-sync
cookie_file | git HTTP cookie file | git-sync

See
[git-sync docs](https://github.com/kubernetes/git-sync/blob/master/docs/ssh.md)
for more information

## Using GitHub

You will need to create SSH credentials to access the GitHub sample GIT
repository, and ensure that those credentials are usable for GitHub access. You
can choose to use any other Git provider instead and set up credentials
accordingly, in which case you can skip this section.

**IMPORTANT NOTE:** In production, it is recommended to use
[deploy keys](https://developer.github.com/v3/guides/managing-deploy-keys/#deploy-keys)
to grant access to a single GitHub repo instead of your personal account.

In a terminal session of your Linux machine issue the following command:

```console
$ ssh-keygen -t rsa -b 4096 -C "your_email@example.com" -N '' -f $HOME/.ssh/id_rsa.nomos
```

This command will create a pair of keys, `$HOME/.ssh.id_rsa.nomos`, and
`$HOME/.ssh.id_rsa.nomos.pub` that will be used to set up git repo access in GKE
Policy Management.

Note that the resulting private key must _not_ be password protected. This key
should not be used for purposes other than this example exercise.

[Upload](https://help.github.com/articles/adding-a-new-ssh-key-to-your-github-account/)
the file `$HOME/.ssh/id_rsa.nomos.pub`, which was generated in the previous
step, to your account on Github. This file will be used by the GKE Policy
anagement installation to access the sample git repository. The file
`$HOME/.ssh/id_rsa.nomos` should be guarded carefully as any other private key
file.

Now, [test](https://help.github.com/articles/testing-your-ssh-connection/) your
SSH connection to github using the key you just generated and populate known
hosts file:

```console
$ ssh -o IdentityAgent=/dev/null -F /dev/null -i $HOME/.ssh/id_rsa.nomos -T git@github.com
Hi <your_username>! You've successfully authenticated, but GitHub does not provide shell access.
```

An example policy hierarchy is available in
[this Github repository](https://github.com/frankfarzan/foo-corp-example).
[Fork this repo on Github](https://help.github.com/articles/fork-a-repo/) or to
your preferred Git hosting provider if you want to make changes.

You can now clone the sample repository locally as follows:

```console
$ ssh-add $HOME/.ssh/id_rsa.nomos
$ git clone git@github.com:frankfarzan/foo-corp-example.git foo
```

or your own copy as:

```console
$ git clone git@github.com:your_github_username/foo-corp-example.git
```

## Next Steps

Consult the [installation guide](installation.md) for instructions on how
to apply the config.
