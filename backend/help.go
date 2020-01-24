package backend

const (
	helpDescriptionBackend = `
The postgres cluster secret engine is used to generate temporary database 
credentials.
`

	helpSynopsisInfo = `
Returns the build information about the plugin.
`

	helpDescriptionInfo = ``

	helpSynopsisCluster = `
Write, Read and Delete cluster configuration.
`

	helpDescriptionCluster = `
Registering a cluster for the first time will automatically change the password
provided in the request. Vault will also create a new role in the cluster and
send back the role name in response. It is strongly recommended that you do
not make any change to the root or management role once the cluster has been
registered.  

Reading from this endpoint should be protected since reading cluster configuration
will send back the password for both root and management users.

Deleting a cluster has no effect on the actual resource. Vault still retains the
configuration for a deleted cluster but the cluster is marked as 'disabled'.
Disabling a cluster prevents creation of new databases or credentials in it and
immediately disables all existing databases.  
It will not be possible to renew the lease on a disabled cluster and any active
lease will be revoked on expiry.

Listing this endpoint lists all active or deleted databases that have been
registered in the cluster so far.
`

	helpSynopsisListClusters = `
List the names of all clusters that have been registered so far
`

	helpDescriptionListClusters = `
List clusters path returns a list of all registered clusters, active or
disabled. Note that the list contains no information about whether the cluster
is active or not.`

	helpSynopsisClone = `
Use existing cluster configuration to configure a new cluster.
`

	helpDescriptionClone = `
Clone endpoint is used to copy the cluster configuration without exposing any
sensitive detail. This endpoint is particularly useful when combined with
snapshot restores or database cloning features provided by most cloud providers.

Cloning a cluster will first use the source credentials to validate the connection
with clone endpoint and, if successful, will rotate the password for both root
and management user. All the other details are kept intact.
`

	helpSynopsisDatabase = `
Write, Read and Delete database configuration.
`

	helpDescriptionDatabase = `
Writing to this endpoint will attempt to create a database with matching name in
the cluster. The request will fail if the database already exist. It is not possible
to write to a database in a cluster that is marked 'disabled'.

Vault will create an owner role for each database. This role will ultimately own
all objects created by temporary users. It is possible to override this behaviour
using role provided creation statements but keep in mind that if the ownership
is not transferred and re-assigned properly then the temporary users will not be
able to use objects created by each other.

This endpoint can not be used to read a delete4d database configuration 
or a database that exists in deleted cluster.

Deleting a database does not drop the actual resource. Vault still retains the
configuration for all deleted databases but prevents any new operation or lease
renewal on it.
`

	helpSynopsisListRoles = `
List the names of all registered roles
`

	helpDescriptionListRoles = ``

	helpSynopsisRoles = `
Write, Read and Delete roles
`

	helpDescriptionRoles = `
A role describes the TTL on credential lease and optionally the queries to create
and revoke the database users.

Creating a new role makes it available to all registered clusters and databases.

Deleting a role does not revoke the credentials derived from it but it does prevent
lease renewal. All active lease on a role will be revoked on expiry.
`

	helpSynopsisCreds = `
Generate temporary credential pair against a role and database.
`

	helpDescriptionCreds = `
This endpoint is used to generate temporary credentials. Credentials can be generated
for any active database in an active cluster.  
The TTL of the lease, query to create role, grant proper permissions to it, and revoke
the role on lease expiry is all decided by the role specified in the request.

If a role is deleted while a lease is still active on it, the lease can no longer be
renewed. In this case the plugin will also use a pre-configured query to revoke the
lease on expiry. Note that in this situation the plugin will revoke the credentials
on best effort basis and if a query fails during cleanup it will be returned as a
response warning rather than an error. In any case the lease will be revoked by vault.

If the 'creation_statements' and 'revocation_statements' parameters are left empty then
the plugin will use following queries to create and drop users.
`

	helpSynopsisMetadata = `Attach arbitrary key-value pairs to cluster or database object`

	helpDescriptionMetadata = `
Metadata endpoint is used to attach arbitrary key-value pairs with a cluster or database.
This metadata can be used to lookup the configured objects.

Key-Value pairs are opaque to Vault or the backend and can be any arbitrary string. The
maximum length of key can be 64 characters and maximum length of value can be 128 characters.
A longer key or value will result in an error response.

When creating metadata "cluster" attribute is always required and must resolve to a valid
cluster already registered with the backend. If "database" attribute is provided then it must
be a valid database registered in given cluster.
When only "cluster" attribute is specified the metadata will be associated with the cluster
only and when both "cluster" and "database" attributes are specified the metadata will be
associated with the database only.
Databases do not inherit the metadata from their prent cluster.

When reading from metadata endpoint "lookup" attribute is required and must be set to either
"database" or "cluster". A positive lookup requires all provided attributes to match, that means
you can perform lookups using a subset of metadata but not using a superset.

Listing metadata endpoint returns a map from object identifier to map of key-values.
`

	helpSynopsisGCListClusters = "List all clusters that are deleted and ready for GC"

	helpDescriptionGCListClusters = `
This endpoint lists all clusters that have been marked as deleted. It is useful for review
and audit before the cluster and databases are purged permanently.`

	helpSynopsisGCClusterOps = "Read, list and purge a cluster configuration"

	helpDescriptionGCClusterOps = `
This endpoint is used to list the databases in a cluster, read cluster configuration
or purge the cluster from vault.

Listing the endpoint returns a list of all deleted databases in the cluster.
Note that the databases that are not marked as deleted will not appear in
response.

Reading the endpoint returns the configuration of a database cluster complete
with its root credentials and owner details. Vault does not track the credentials
after the cluster is marked as deleted so the credentials returned by this
endpoint are the ones that vault had known before the cluster was deleted.

Deleting the cluster from this endpoint purges the cluster information from
vault and the cluster name becomes available for use once again.
When a cluster is purged, all of its databases are also purged from vault.
Note that vault does not attempt to drop the databases from the physical postgres
cluster, it only deletes the configuration from its own storage.

A cluster can only be deleted using this endpoint if it is marked as deleted.
`

	helpSynopsisGCDbOps = "Read and delete database configuration from a cluster"

	helpDescriptionGCDbOps = `
This endpoint is used to read and delete database configuration from a cluster.

Reading from this endpoint returns the database configuration along with its
owner username and password. A database can only be read from this endpoint if
it has been marked as deleted.

Deleting from this endpoint deleted the database configuration from vault
storage. Note that vault does not attempt to drop the database from physical
postgres cluster, it only deleted the configuration from its own storage.

A database can only be deleted using this endpoint if it is marked as deleted.
`
)
