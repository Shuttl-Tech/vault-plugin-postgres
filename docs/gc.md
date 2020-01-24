    Request:        gc/clusters
    Matching Route: ^gc/clusters$

List all clusters that are deleted and ready for GC


## DESCRIPTION

This endpoint lists all clusters that have been marked as deleted. It is useful for review
and audit before the cluster and databases are purged permanently.

---

    Request:        gc/cluster/c
    Matching Route: ^gc/cluster/(?P<cluster>\w(([\w-.]+)?\w)?)$

Read, list and purge a cluster configuration

## PARAMETERS

    cluster (string)
        Name of the database cluster

## DESCRIPTION

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

---

    Request:        gc/cluster/c/d
    Matching Route: ^gc/cluster/(?P<cluster>\w(([\w-.]+)?\w)?)/(?P<database>\w(([\w-.]+)?\w)?)$

Read and delete database configuration from a cluster

## PARAMETERS

    cluster (string)
        Name of the database cluster

    database (string)
        Name of the database

## DESCRIPTION

This endpoint is used to read and delete database configuration from a cluster.

Reading from this endpoint returns the database configuration along with its
owner username and password. A database can only be read from this endpoint if
it has been marked as deleted.

Deleting from this endpoint deleted the database configuration from vault
storage. Note that vault does not attempt to drop the database from physical
postgres cluster, it only deleted the configuration from its own storage.

A database can only be deleted using this endpoint if it is marked as deleted.


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
