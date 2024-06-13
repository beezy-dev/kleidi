# kubernetes kms plugin

## Physical/Virtual machine world
The entire IT organization is segmented into knowledge domains such as networking, storage, computing, and applications in the legacy world. 
When an application team asks for a virtual machine:
* The VMware Team has its credentials, which the Linux team cannot access.
* The Linux team will configure, maintain, and support the operating system and not share their credentials with any other team. 
* The application team will deploy their application, which might be connected to a database; the DBA will provide credentials.   
This quick overview can be enriched with all other layers like storage, backup, networking, monitoring, etc.
None will cross-share their credentials.

## Container platform world
Within Kubernetes, the states and configurations of every component, from computing to networking to applications and more, are stored within the ```etcd``` key-value datastore. 

Even if cloud-native applications can interact directly with a KMS provider like Vault, application and platform credentials are still stored within the cluster. This might also include the token to connect with the KMS provider.

All data fields are encoded in base64 but not encrypted. 

## Security exposures

The following diagram takes a 10,000-feet overview to explore the security exposures leading to a potential secret leaking/spilling: 

![kleidi security exposures](images/kledi-security_exposure.drawio.png)

* The secret comes from an external source and needs to be injected.  
* The base64 encoded secret will be ingested via the API server. 
* If a Kubernetes KMS provider plugin exists, the API server encrypts the data field using an envelope encryption scheme. 
* The secret and encrypted data filed will be stored within the ```etcd``` key-value datastore. 
* The ```etcd``` key-value datastore file is saved on a local volume on the control plane node filesystem. 

What are the exposures:

|Exposure | Risk | Mitigation |
|---------|------|------------|
|The secret comes from an external source. It requires a base64-encoded payload. | This transformation is a first-level exposure of the data field at file and console levels. | Work an injection mechanism from the password manager or KMS |  
| A common mistake is committing the secret YAML definition to a Git repository. | The credentials are exposed for life and will need to be rotated. | Don't include any YAML manifest with sensitive data in a Git repository even for testing purposes. Using a tool like SOPS can help prevent such scenario | 
| If no KMS provider plugin exists. | The API server stores the base64-encoded secret within the ```etcd``` key-value datastore. | Application secrets might benefit from an external KMS. Platform secrets will require a data encryption at rest option provided by Kubernetes. |
| If a KMS provider plugin exists. | The encryption Key or credentials to access the KMS or HSM are exposed in clear text. | Set up a mTLS authentication if possible. |
| The ```etcd``` key-value datastore is stored on the control plane filesystem. | The datastore file can be accessed if the node is compromised. | Encrypting the filesystem helps secure the datastore file from being read, except if the node has been compromised with root access. |
| The API server is the Kubernetes heart and soul. | If the appropriate RBAC or the API server is compromised, all protective measures will be useless since the API server will decrypt all sensitive data fields. | RBAC and masking the API server if possible | 

Thanks to Red Hat colleagues Francois Duthilleul and Frederic Herrmann for spending time analyzing the gaps.

