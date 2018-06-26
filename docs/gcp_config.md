# GCP Configuration

**NOTE: GCP support is not yet available.**

A sample YAML file for configuring GCP-based Nomos is provided below:

```yaml
contexts:
  - kubeconfig_context_of_your_cluster
gcp:
  ORG_ID: 515925372711
  PRIVATE_KEY_FILENAME: $HOME/private_key.json
user: youruser@foo-corp.com
```

Note:

*   `contexts` field is a list of clusters where Nomos will be installed. Run
    `kubectl config get-contexts` to see what contexts are available to you.
*   Set `user` field to be set to your username that is valid for authenticating
    to the clusters. This username must be valid on all clusters included in the
    contexts field.
*   Please do NOT replace $HOME with the name of your actual home directory,
    leave it as a placeholder.

These are all the supported keys for the the gcp object of the installer config
file.

Key                  | Description
-------------------- | -----------
ORG_ID               | [GCP organization id](https://cloud.google.com/resource-manager/docs/creating-managing-organization#retrieving_your_organization_id)
PRIVATE_KEY_FILENAME | Path to the file containing the [GCP service account private key](#creating-service-account) used for accessing GCP Kubernetes Policy API.

## Config Reference

This section enumerates ConfigMaps and Secrets used by Nomos. When using
installer, these are automatically created in `nomos-system` namespace.

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

## Creating Service Account

1.  [Create a service account][1]
2.  [Grant the service account][2] `Kubernetes Policy Viewer` role
3.  [Create a servie account key][3] and download the JSON private key.

[1]: https://cloud.google.com/iam/docs/creating-managing-service-accounts
[2]: https://cloud.google.com/iam/docs/granting-roles-to-service-accounts
[3]: https://cloud.google.com/iam/docs/creating-managing-service-account-keys
