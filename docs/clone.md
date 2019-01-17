Request:        clone/name
Matching Route: ^clone/(?P<cluster>\w(([\w-.]+)?\w)?)$

Use existing cluster configuration to configure a new cluster.

## PARAMETERS

    cluster (string)
        Identifier of the source cluster to clone

    host (string)
        Host name of the target cluster

    inherit_deleted_db (bool)
        If set to true the clone will inherit the configuration for deleted databases as well

    port (int)
        Port number of the target cluster

    target (string)
        Identifier for the target cluster

## DESCRIPTION

Clone endpoint is used to copy the cluster configuration without exposing any
sensitive detail. This endpoint is particularly useful when combined with
snapshot restores or database cloning features provided by most cloud providers.

Cloning a cluster will first use the source credentials to validate the connection
with clone endpoint and, if successful, will rotate the password for both root
and management user. All the other details are kept intact.
