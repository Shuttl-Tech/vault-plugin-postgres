    Request:        creds/c/d/r
    Matching Route: ^creds/(?P<cluster>\w(([\w-.]+)?\w)?)/(?P<database>\w(([\w-.]+)?\w)?)/(?P<role>\w(([\w-.]+)?\w)?)$

Generate temporary credential pair against a role and database.

## PARAMETERS

    cluster (string)
        Name of the cluster

    database (string)
        Name of the database in cluster

    role (string)
        Name of the role

## DESCRIPTION

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
