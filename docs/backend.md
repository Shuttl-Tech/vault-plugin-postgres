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

    ^cluster/(?P<cluster>\w(([\w-.]+)?\w)?)$
        Write, Read and Delete cluster configuration.

    ^cluster/(?P<cluster>\w(([\w-.]+)?\w)?)/(?P<database>\w(([\w-.]+)?\w)?)$
        Write, Read and Delete database configuration.

    ^creds/(?P<cluster>\w(([\w-.]+)?\w)?)/(?P<database>\w(([\w-.]+)?\w)?)/(?P<role>\w(([\w-.]+)?\w)?)$
        Generate temporary credential pair against a role and database.

    ^info$
        Returns the build information about the plugin.

    ^roles/(?P<name>\w(([\w-.]+)?\w)?)$
        Write, Read and Delete roles

    ^roles/?$
        List the names of all registered roles
