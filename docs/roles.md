    Request:        roles/name
    Matching Route: ^roles/(?P<name>\w(([\w-.]+)?\w)?)$

Write, Read and Delete roles

## PARAMETERS

    creation_statement (slice)
        Database statements to create and configure a user

    default_ttl (duration (sec))
        TTL for the lease associated with this role

    max_ttl (duration (sec))
        Maximum TTL for the lease associated with this role

    name (string)
        Unique identifier for the role

    revocation_statement (slice)
        Database statements to drop a user and revoke permissions

## DESCRIPTION

A role describes the TTL on credential lease and optionally the queries to create
and revoke the database users.

Creating a new role makes it available to all registered clusters and databases.

Deleting a role does not revoke the credentials derived from it but it does prevent
lease renewal. All active lease on a role will be revoked on expiry.

---

    Request:        roles
    Matching Route: ^roles/?$

List the names of all registered roles


## DESCRIPTION

<no description>


### TOC

 - [clone](./clone.md)
 - [cluster](./cluster.md)
 - [creds](./creds.md)
 - [database](./database.md)
 - [index](./index.md)
 - [info](./info.md)
 - [metadata](./metadata.md)
 - [roles](./roles.md)
