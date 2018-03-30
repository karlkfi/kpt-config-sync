# Nomos Quickstart

This quickstart shows a step-by-step guide to install Nomos on Kubernetes clusters and create and synchronize hierarchical policies from a Git repository.

To understand the installation steps in more detail, please refer to the [Nomos User Guide](user_guide.md).


## Before you begin

Before installing Nomos, there are a few tasks that must be performed once to ensure that your work environment is complete and able to support the installation.

This setup takes about 30 minutes.


### Linux

You will need a Linux environment to drive the installation.

Currently the only tested environment for installation is Linux on amd64.  The following utilities need to be installed: `docker`, `bash`, `curl`, and `gcloud`.


### Kubernetes

You will need to have at least one up-and-running Kubernetes cluster. The cluster version **must be** 1.9 or above. You must have credentials available to access this cluster stored in your local kubectl config file (usually at `~/.kube/config`).

The easiest way to get all of these is to follow the [GKE quick start guide](https://cloud.google.com/kubernetes-engine/docs/quickstart) and make sure to select version 1.9+ when creating the cluster.


### GitHub Repo

You will need to create SSH credentials to access the GitHub sample git repository, and ensure that those credentials are usable for GithHub access. You can choose to use any other Git provider instead and set up credentials accordingly, in which case you can skip this section.

**IMPORTANT NOTE:** In production, it is recommended to use [deploy keys](https://developer.github.com/v3/guides/managing-deploy-keys/#deploy-keys) to grant access to a single GitHub repo instead of your personal account.

In a terminal session of your Linux machine issue the following command:


```
ssh-keygen -t rsa -b 4096 -C "your_email@example.com" -N '' -f $HOME/.ssh/id_rsa.nomos
```


This command will create a pair of keys, `$HOME/.ssh.id_rsa.nomos`, and `$HOME/.ssh.id_rsa.nomos.pub` that will be used to set up git repo access in Nomos.  

Note that the resulting private key must _not_ be password protected. This key should not be used for purposes other than this example exercise.

[Upload](https://help.github.com/articles/adding-a-new-ssh-key-to-your-github-account/) the file `$HOME/.ssh/id_rsa.nomos.pub`, which was generated in the previous step, to your account on Github. This file will be used by the Nomos installation to access the sample git repository.  The file `$HOME/.ssh/id_rsa.nomos` should be guarded carefully as any other private key file.

Now, [test](https://help.github.com/articles/testing-your-ssh-connection/) your SSH connection to github using the key you just generated.


```
$ ssh -o IdentityAgent=/dev/null -F /dev/null -i $HOME/.ssh/id_rsa.nomos -T git@github.com
Hi <your_username>! You've successfully authenticated, but GitHub does not provide shell access.
```


An example policy hierarchy is available in [this Github repository](https://github.com/frankfarzan/foo-corp-example). [Fork this repo on Github](https://help.github.com/articles/fork-a-repo/) or to your preferred Git hosting provider if you want to make changes.

You can now clone the sample repository locally as follows:


```
$ ssh-add $HOME/.ssh/id_rsa.nomos
$ git clone git@github.com:frankfarzan/foo-corp-example.git foo
```


or your own copy as:


```
$ git clone git@github.com:your_github_username/foo-corp-example.git
```



### Installer

Download the Nomos installer script to a directory on your machine.


```
cd
mkdir -p tmp/nomos
cd tmp/nomos
curl https://storage.googleapis.com/nomos-release/run-installer.sh -o run-installer.sh
chmod +x run-installer.sh
```



## Installation

The installer can run in two modes: batch (the default) and interactive (menu-driven).


### Batch installation

Batch installation is based on reading a prepared configuration file. A sample configuration is provided below. Most fields have been preset to point at the sample configuration repository, but some you must supply, as noted below. 

Note:



*   For the meanings of all fields appearing in the configuration file, please refer to the [Nomos Open Source User Guide](https://drive.google.com/a/google.com/open?id=1vZIrA1GKfD9NmIq0dPGfzF-S6q2RxnRxImSR0vE_FKI).
*   You must edit the `contexts` field to point at the list of clusters you want to install on.  Use `kubectl config get-contexts` to see what contexts are available to you.
*   You must edit the `user` field to be set to your username that is valid for authenticating to the clusters.  This username must be valid on all clusters included in the `contexts` field.
*   Please do **NOT** replace `$HOME` with the name of your actual home directory, leave it as a placeholder.
*   Update `GYT_SYNC_REPO` to point to your repo if you're not using the sample repo rrovided

```
contexts:
- kubeconfig_context_of_your_cluster
git:
  GIT_SYNC_BRANCH: master
  GIT_SYNC_REPO: git@github.com:frankfarzan/foo-corp-example.git
  GIT_SYNC_WAIT: 60
  ROOT_POLICY_DIR: foo-corp
ssh:
  knownHostsFilename: $HOME/.ssh/known_hosts
  privateKeyFilename: $HOME/.ssh/id_rsa.nomos
user: youruser@example.com
```


Once you have finished with editing the file, you can run the batch installer as follows:


```
./run-installer.sh --config=/path/to/your/config.yaml
```



### Interactive installation

Interactive installation is menu driven.  It allows you to edit the configuration through a user-friendly menu as shown in the figure below.


![drawing](img/installer_interactive.png)


You can start from a sample configuration:


```
./run-installer.sh --interactive
```


or you can start from an existing configuration, similar to the one in the [Batch Installation](#batch-installation) section.


```
./run-installer.sh --interactive --config=/path/to/your/config.yaml
```


"Save" will store the current settings that you can then reuse later in a batch installation. "Install" will run the installer on the chosen clusters.


### Verify installation

To verify that Nomos components are correctly installed, issue the following command and verify that all components listed have status displayed as "Running."

Check running components:


```
$ kubectl get pods -n=nomos-system
NAME                                                  READY     STATUS    RESTARTS   AGE
git-policy-importer-66bf6b9db4-pbsxn                  2/2       Running   0          24m
resourcequota-admission-controller-64988d97f4-nxmsc   1/1       Running   0          24m
syncer-58545bc77d-l485n                               1/1       Running   0          24m
```


Check present namespaces:


```
$ kubectl get ns
NAME               STATUS    AGE
audit              Active    2m
default            Active    2m
kube-public        Active    2m
kube-system        Active    2m
shipping-dev       Active    2m
shipping-prod      Active    2m
shipping-staging   Active    2m
nomos-system       Active    2m

```



## Creating hierarchical policies

Once Nomos components are deployed and running in a cluster, namespaces will be automatically created:


```
kubectl get namespaces -l nomos-managed
```


This should return 4 namespaces: `shipping-dev`, `shipping-staging`,` shipping-prod`, and `audit`.

Rolebindings are inherited from parent directories:


```
kubectl get rolebinding -n shipping-dev
```


This should return 3 rolebindings: `job-creators`, `pod-creators`, and `viewers`.

You can test effective RBAC policies by impersonating users. For example, this should be forbidden:


```
kubectl get secrets -n shipping-dev --as bob@foo-corp.com
```


whereas, this should succeed since `bob@foo-corp.com` has the `pod-creator` role:


```
kubectl get pods -n shipping-dev --as bob@foo-corp.com
```


To see inherited ResourceQuota in action, we can create a pod and request resources that exceed the limits set in the parent directory:


```
cat <<EOF | kubectl create --as bob@foo-corp.com -f -
apiVersion: v1
kind: Pod
metadata:
  name: busybox-sleep
  namespace: shipping-prod
spec:
  containers:
  - name: busybox
    image: busybox
    args:
    - sleep
    - "1000000"
    resources:
      requests:
        memory: "64Mi"
        cpu: "2"
EOF
```


Try changing the cpu request from `2` to `200m`. This time it should succeed.


## What's next

[Nomos User Guide](user_guide.md)


## Send feedback

Questions or feedback? Get in touch with us at [nomos-support@google.com](mailto:nomos-support@google.com).

