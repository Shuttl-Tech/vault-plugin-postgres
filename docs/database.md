Request:        cluster/c/database
Matching Route: ^cluster/(?P<cluster>\w(([\w-.]+)?\w)?)/(?P<database>\w(([\w-.]+)?\w)?)$

Write, Read and Delete database configuration.

## PARAMETERS

    cluster (string)
        Name of the cluster in which the new database will be created

    database (string)
        Name of the new database to create

## DESCRIPTION

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
