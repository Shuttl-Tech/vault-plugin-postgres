    Request:        metadata
    Matching Route: ^metadata/?$

Attach arbitrary key-value pairs to cluster or database object

## PARAMETERS

    cluster (string)
        Name of the cluster

    data (keypair)
        key-value pairs to associate with the object

    database (string)
        Name of the database

    type (string)
        Type of the object to lookup. Must be one of 'cluster' or 'database'

## DESCRIPTION

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


### TOC

 - [clone](./docs/clone.md)
 - [cluster](./docs/cluster.md)
 - [creds](./docs/creds.md)
 - [database](./docs/database.md)
 - [index](./docs/index.md)
 - [info](./docs/info.md)
 - [metadata](./docs/metadata.md)
 - [roles](./docs/roles.md)
