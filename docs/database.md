    Request:        cluster/c/database
    Matching Route: ^cluster/(?P<cluster>\w(([\w-.]+)?\w)?)/(?P<database>\w(([\w-.]+)?\w)?)$

Write, Read and Delete database configuration.

## PARAMETERS

    cluster (string)
        Name of the cluster in which the new database will be created

    create_db (bool)
        If true vault will create new database in cluster

    database (string)
        Name of the new database to create

    initialize (bool)
        If true vault will create necessary roles in cluster

    objects_owner_role (string)
        Role that will own all objects in this database

## DESCRIPTION

Writing to this endpoint will attempt to create a database with matching name in
the cluster. The request will fail if the database already exist. It is not possible
to write to a database in a cluster that is marked 'disabled'.

Vault will create an owner role for each database. This role will ultimately own
all objects created by temporary users. It is possible to override this behaviour
using role provided creation statements but keep in mind that if the ownership
is not transferred and re-assigned properly then the temporary users will not be
able to use objects created by each other.

This endpoint can not be used to read a deleted database configuration 
or a database that exists in deleted cluster.

Deleting a database does not drop the actual resource. Vault still retains the
configuration for all deleted databases but prevents any new operation or lease
renewal on it.


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
