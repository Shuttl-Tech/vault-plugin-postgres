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
)
