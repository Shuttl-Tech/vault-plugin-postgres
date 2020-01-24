## DESCRIPTION

The postgres cluster secret engine is used to generate temporary database 
credentials.

## PATHS

The following paths are supported by this backend. To view help for
any of the paths below, use the help command with any route matching
the path pattern. Note that depending on the policy of your auth token,
you may or may not be able to access certain paths.

    ^clone/(?P<cluster>\w(([\w-.]+)?\w)?)$
        Use existing cluster configuration to configure a new cluster.

    ^cluster/(?P<cluster>\w(([\w-.]+)?\w)?)/(?P<database>\w(([\w-.]+)?\w)?)$
        Write, Read and Delete database configuration.

    ^cluster/(?P<cluster>\w(([\w-.]+)?\w)?)/?$
        Write, Read and Delete cluster configuration.

    ^cluster/?$
        List the names of all clusters that have been registered so far

    ^creds/(?P<cluster>\w(([\w-.]+)?\w)?)/(?P<database>\w(([\w-.]+)?\w)?)/(?P<role>\w(([\w-.]+)?\w)?)$
        Generate temporary credential pair against a role and database.

    ^gc/cluster/(?P<cluster>\w(([\w-.]+)?\w)?)$
        Read, list and purge a cluster configuration

    ^gc/cluster/(?P<cluster>\w(([\w-.]+)?\w)?)/(?P<database>\w(([\w-.]+)?\w)?)$
        Read and delete database configuration from a cluster

    ^gc/clusters$
        List all clusters that are deleted and ready for GC

    ^info$
        Returns the build information about the plugin.

    ^metadata/(?P<id>.+)$
        Delete metadata using ID

    ^metadata/?$
        Attach arbitrary key-value pairs to cluster or database object

    ^roles/(?P<name>\w(([\w-.]+)?\w)?)$
        Write, Read and Delete roles

    ^roles/?$
        List the names of all registered roles


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
