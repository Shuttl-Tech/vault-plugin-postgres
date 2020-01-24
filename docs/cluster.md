    Request:        cluster
    Matching Route: ^cluster/?$

List the names of all clusters that have been registered so far


## DESCRIPTION

List clusters path returns a list of all registered clusters, active or
disabled. Note that the list contains no information about whether the cluster
is active or not.

---

    Request:        cluster/name
    Matching Route: ^cluster/(?P<cluster>\w(([\w-.]+)?\w)?)/?$

Write, Read and Delete cluster configuration.

## PARAMETERS

    cluster (string)
        Name of the cluster

    database (string)
        Name of the database to connect with

    host (string)
        Host name of the writer node in cluster

    max_connection_lifetime (duration (sec))
        Maximum time duration to keep an idle connection alive

    max_idle_connections (int)
        Maximum number of idle connections to maintain in pool

    max_open_connections (int)
        Maximum number of connections to create with the database

    password (string)
        Password to authenticate the database connection

    port (int)
        Port number to be used for database connections

    ssl_mode (string)
        Whether or not to use SSL

    username (string)
        Username that can be used to connect with the cluster

## DESCRIPTION

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


### TOC

 - [clone](./clone.md)
 - [cluster](./cluster.md)
 - [creds](./creds.md)
 - [database](./database.md)
 - [gc](./gc.md)
 - [index](./index.md)
 - [info](./info.md)
 - [metadata](./metadata.md)
 - [roles](./roles.md)
