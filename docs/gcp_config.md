# Google Cloud Platform Configuration

**NOTE: This is an alpha release of Kubernetes Policy API and available to
whitelisted customers.**

A sample YAML file for configuring GCP-based GKE Policy Management is provided
below:

```yaml
contexts:
  - kubeconfig_context_of_your_cluster
gcp:
  ORG_ID: 515925372711
  PRIVATE_KEY_FILENAME: $HOME/private_key.json
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
*   Only files in the home directory (as resolved by `$HOME`) can be specified
    in this config file and in kubectl config for the context.

These are all the supported keys for the the gcp object of the installer config
file.

Key                  | Description
-------------------- | -----------
ORG_ID               | [GCP organization id](https://cloud.google.com/resource-manager/docs/creating-managing-organization#retrieving_your_organization_id)
PRIVATE_KEY_FILENAME | Path to the file containing the [GCP service account private key](#obtaining-private-key) used for accessing GCP Kubernetes Policy API.

## Obtaining Private Key

1.  [Create a service account][1]
2.  [Grant the service account][2] `Kubernetes Policy Viewer` role
    *   Currently the `Kubernetes Policy Viewer` role is not visible from the
        Google Cloud Console. You must use the gcloud CLI to make the change,
        see the example below.
3.  [Create a service account key][3] and download the JSON private key.

A sample script that creates a service account named `policy-viewer-sa` in
project `my-sa-project` is provided below:

```console
gcloud iam service-accounts create policy-viewer-sa \
    --display-name "GKE Policy Retriever Account" --project my-sa-project

gcloud organizations add-iam-policy-binding 515925372711 \
    --member serviceAccount:policy-viewer-sa@my-sa-project.iam.gserviceaccount.com \
    --role roles/kubernetespolicy.policyViewer

gcloud iam service-accounts keys create ~/private_key.json \
    --iam-account policy-viewer-sa@my-sa-project.iam.gserviceaccount.com
```

Be sure to enable the API in `my-sa-project` before installation:

```console
$ gcloud services enable kubernetespolicy.googleapis.com --project my-sa-project
```

## Config Reference

This section enumerates ConfigMaps and Secrets used by GKE Policy Management.
When using installer, these are automatically created in `nomos-system`
namespace.

### configmap/gcp-policy-importer

Used by gcppolicyimporter deployment:

Key                         | Description
--------------------------- | -----------
ORG_ID                      | [GCP organization id](https://cloud.google.com/resource-manager/docs/creating-managing-organization#retrieving_your_organization_id)
GOOGLE_GCP_CREDENTIALS_FILE | Path of the file containing the GCP service account private key used for accessing GCP Kubernetes Policy API.
GCP_POLICY_IMPORTER_CA_FILE | Path of the Root CA certificate file to use in place of the system one; for testing only.
POLICY_API_ADDRESS          | Kubernetes Policy API address; for testing only.

### secret/gcp-creds

Used by gcppolicyimporter deployment:

Key             | Description
--------------- | -------------------------------
gcp-private-key | GCP service account private key

[1]: https://cloud.google.com/iam/docs/creating-managing-service-accounts
[2]: https://cloud.google.com/iam/docs/granting-roles-to-service-accounts
[3]: https://cloud.google.com/iam/docs/creating-managing-service-account-keys
