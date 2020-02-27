package backend

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"time"
)

type Path string

const (
	PathCluster  Path = "config/cluster/%s"
	PathDatabase Path = "config/cluster/%s/database/%s"
	PathRole     Path = "config/role/%s"
	PathMeta     Path = "meta/%s"
)

const (
	queryCreateDb               = `create database {{database}}`
	queryCreateObjectsOwnerRole = `create role {{role_name}} role {{role_group_management}}, {{role_group_root}}`
	queryGrantAll               = `grant all privileges on all tables in schema public to {{role_name}}`
	queryUpdatePassword         = `alter user {{user}} with password '{{password}}'`
	queryCreateManagementRole   = `create role {{user}} with login password '{{password}}' createrole nocreatedb noinherit`
	queryRenewExpiry            = `alter role {{user}} valid until '{{expiration}}'`
)

const SecretCredsType = "creds"

var ErrNotFound = errors.New("requested record was not found")

func (p Path) For(n ...interface{}) string {
	return fmt.Sprintf(string(p), n...)
}

type backend struct {
	*framework.Backend
}

func Factory(ctx context.Context, c *logical.BackendConfig) (logical.Backend, error) {
	b := New(c)
	if err := b.Setup(ctx, c); err != nil {
		return nil, err
	}
	return b, nil
}

func New(c *logical.BackendConfig) *backend {
	b := backend{}

	b.Backend = &framework.Backend{
		BackendType: logical.TypeLogical,
		PathsSpecial: &logical.Paths{
			Unauthenticated: []string{"info"},
		},
		Secrets: []*framework.Secret{
			{
				Type: SecretCredsType,
				Fields: map[string]*framework.FieldSchema{
					"username": {
						Type:        framework.TypeString,
						Description: "Username",
					},
					"password": {
						Type:        framework.TypeString,
						Description: "Password",
					},
				},
				Renew:  b.secretCredsRenew,
				Revoke: b.secretCredsRevoke,
			},
		},
		Paths: []*framework.Path{
			{
				Pattern: "info",
				Operations: map[logical.Operation]framework.OperationHandler{
					logical.ReadOperation: NewOperationHandler(b.pathInfo, propsPathInfo),
				},
				HelpSynopsis:    helpSynopsisInfo,
				HelpDescription: helpDescriptionInfo,
			},
			{
				Pattern: "metadata/?$",
				Fields: map[string]*framework.FieldSchema{
					"cluster": {
						Type:        framework.TypeString,
						Description: "Name of the cluster",
					},
					"database": {
						Type:        framework.TypeString,
						Description: "Name of the database",
					},
					"type": {
						Type:        framework.TypeString,
						Description: "Type of the object to lookup. Must be one of 'cluster' or 'database'",
					},
					"data": {
						Type:        framework.TypeKVPairs,
						Description: "key-value pairs to associate with the object",
					},
				},
				Operations: map[logical.Operation]framework.OperationHandler{
					logical.UpdateOperation: NewOperationHandler(b.pathMetadataUpdate, propsMetadataUpdate),
					logical.ReadOperation:   NewOperationHandler(b.pathMetadataRead, propsMetadataRead),
					logical.ListOperation:   NewOperationHandler(b.pathMetadataList, propsMetadataList),
				},
				HelpSynopsis:    helpSynopsisMetadata,
				HelpDescription: helpDescriptionMetadata,
			},
			{
				Pattern: "metadata/(?P<id>.+)",
				Fields: map[string]*framework.FieldSchema{
					"id": {
						Type:        framework.TypeString,
						Description: "Metadata ID",
					},
				},
				Operations: map[logical.Operation]framework.OperationHandler{
					logical.DeleteOperation: NewOperationHandler(b.pathMetadataDelete, propsMetadataDelete),
				},
				HelpSynopsis: "Delete metadata using ID",
			},
			{
				Pattern: "cluster/?$",
				Operations: map[logical.Operation]framework.OperationHandler{
					logical.ListOperation: NewOperationHandler(b.pathClustersList, propsClustersList),
				},
				HelpSynopsis:    helpSynopsisListClusters,
				HelpDescription: helpDescriptionListClusters,
			},
			{
				Pattern: "cluster/" + framework.GenericNameRegex("cluster") + "/?$",
				Fields: map[string]*framework.FieldSchema{
					"cluster": {
						Type:        framework.TypeString,
						Description: "Name of the cluster",
					},
					"host": {
						Type:        framework.TypeString,
						Description: "Host name of the writer node in cluster",
					},
					"port": {
						Type:        framework.TypeInt,
						Description: "Port number to be used for database connections",
						Default:     5432,
					},
					"username": {
						Type:        framework.TypeString,
						Description: "Username that can be used to connect with the cluster",
					},
					"password": {
						Type:        framework.TypeString,
						Description: "Password to authenticate the database connection",
					},
					"max_open_connections": {
						Type:        framework.TypeInt,
						Description: "Maximum number of connections to create with the database",
						Default:     5,
					},
					"max_idle_connections": {
						Type:        framework.TypeInt,
						Description: "Maximum number of idle connections to maintain in pool",
						Default:     5,
					},
					"max_connection_lifetime": {
						Type:        framework.TypeDurationSecond,
						Description: "Maximum time duration to keep an idle connection alive",
						Default:     "300s",
					},
					"database": {
						Type:        framework.TypeString,
						Description: "Name of the database to connect with",
						Default:     "postgres",
					},
					"ssl_mode": {
						Type:        framework.TypeString,
						Description: "Whether or not to use SSL",
						Default:     "require",
					},
				},
				Operations: map[logical.Operation]framework.OperationHandler{
					logical.ReadOperation:   NewOperationHandler(b.pathClusterRead, propsClusterRead),
					logical.UpdateOperation: NewOperationHandler(b.pathClusterUpdate, propsClusterUpdate),
					logical.DeleteOperation: NewOperationHandler(b.pathClusterDelete, propsClusterDelete),
					logical.ListOperation:   NewOperationHandler(b.pathDatabasesList, propsDatabasesList),
				},
				HelpSynopsis:    helpSynopsisCluster,
				HelpDescription: helpDescriptionCluster,
			},
			{
				Pattern: "clone/" + framework.GenericNameRegex("cluster"),
				Fields: map[string]*framework.FieldSchema{
					"cluster": {
						Type:        framework.TypeString,
						Description: "Identifier of the source cluster to clone",
					},
					"target": {
						Type:        framework.TypeString,
						Description: "Identifier for the target cluster",
					},
					"host": {
						Type:        framework.TypeString,
						Description: "Host name of the target cluster",
					},
					"port": {
						Type:        framework.TypeInt,
						Default:     5432,
						Description: "Port number of the target cluster",
					},
					"inherit_deleted_db": {
						Type:        framework.TypeBool,
						Default:     false,
						Description: "If set to true the clone will inherit the configuration for deleted databases as well",
					},
				},
				Operations: map[logical.Operation]framework.OperationHandler{
					logical.UpdateOperation: NewOperationHandler(b.pathCloneUpdate, propsCloneUpdate),
				},
				HelpSynopsis:    helpSynopsisClone,
				HelpDescription: helpDescriptionClone,
			},
			{
				Pattern: "cluster/" + framework.GenericNameRegex("cluster") + "/" + framework.GenericNameRegex("database"),
				Fields: map[string]*framework.FieldSchema{
					"cluster": {
						Type:        framework.TypeString,
						Description: "Name of the cluster in which the new database will be created",
					},
					"database": {
						Type:        framework.TypeString,
						Description: "Name of the new database to create",
					},
					"objects_owner_role": {
						Type:        framework.TypeString,
						Description: "Role that will own all objects in this database",
					},
					"initialize": {
						Type:        framework.TypeBool,
						Description: "If true vault will create necessary roles in cluster",
						Default:     true,
					},
					"create_db": {
						Type:        framework.TypeBool,
						Description: "If true vault will create new database in cluster",
						Default:     true,
					},
				},
				Operations: map[logical.Operation]framework.OperationHandler{
					logical.UpdateOperation: NewOperationHandler(b.pathDatabaseUpdate, propsDatabaseUpdate),
					logical.ReadOperation:   NewOperationHandler(b.pathDatabaseRead, propsDatabaseRead),
					logical.DeleteOperation: NewOperationHandler(b.pathDatabaseDelete, propsDatabaseDelete),
				},
				HelpSynopsis:    helpSynopsisDatabase,
				HelpDescription: helpDescriptionDatabase,
			},
			{
				Pattern: "roles/?$",
				Operations: map[logical.Operation]framework.OperationHandler{
					logical.ListOperation: NewOperationHandler(b.pathRoleList, propsRoleList),
				},
				HelpSynopsis:    helpSynopsisListRoles,
				HelpDescription: helpDescriptionListRoles,
			},
			{
				Pattern: "roles/" + framework.GenericNameRegex("name"),
				Fields: map[string]*framework.FieldSchema{
					"name": {
						Type:        framework.TypeString,
						Description: "Unique identifier for the role",
					},
					"default_ttl": {
						Type:        framework.TypeDurationSecond,
						Description: "TTL for the lease associated with this role",
						Default:     c.System.DefaultLeaseTTL(),
					},
					"max_ttl": {
						Type:        framework.TypeDurationSecond,
						Description: "Maximum TTL for the lease associated with this role",
						Default:     c.System.MaxLeaseTTL(),
					},
					"creation_statement": {
						Type:        framework.TypeStringSlice,
						Description: "Database statements to create and configure a user",
						Default:     defaultCreationSQL,
					},
					"revocation_statement": {
						Type:        framework.TypeStringSlice,
						Description: "Database statements to drop a user and revoke permissions",
						Default:     defaultRevocationSQL,
					},
				},
				Operations: map[logical.Operation]framework.OperationHandler{
					logical.UpdateOperation: NewOperationHandler(b.pathRoleUpdate, propsRoleUpdate),
					logical.ReadOperation:   NewOperationHandler(b.pathRoleRead, propsRoleRead),
					logical.DeleteOperation: NewOperationHandler(b.pathRoleDelete, propsRoleDelete),
				},
				HelpSynopsis:    helpSynopsisRoles,
				HelpDescription: helpDescriptionRoles,
			},
			{
				Pattern: "creds/" + framework.GenericNameRegex("cluster") + "/" + framework.GenericNameRegex("database") + "/" + framework.GenericNameRegex("role"),
				Fields: map[string]*framework.FieldSchema{
					"cluster": {
						Type:        framework.TypeString,
						Description: "Name of the cluster",
					},
					"database": {
						Type:        framework.TypeString,
						Description: "Name of the database in cluster",
					},
					"role": {
						Type:        framework.TypeString,
						Description: "Name of the role",
					},
				},
				Operations: map[logical.Operation]framework.OperationHandler{
					logical.ReadOperation: NewOperationHandler(b.secretCredsCreate, propsCredsRead),
				},
				HelpSynopsis:    helpSynopsisCreds,
				HelpDescription: helpDescriptionCreds,
			},
			{
				Pattern: "gc/clusters",
				Operations: map[logical.Operation]framework.OperationHandler{
					logical.ListOperation: NewOperationHandler(b.gcListClusters, propsGcListClusters),
				},
				HelpSynopsis:    helpSynopsisGCListClusters,
				HelpDescription: helpDescriptionGCListClusters,
			},
			{
				Pattern: "gc/cluster/" + framework.GenericNameRegex("cluster"),
				Fields: map[string]*framework.FieldSchema{
					"cluster": {
						Type:        framework.TypeString,
						Description: "Name of the database cluster",
					},
				},
				Operations: map[logical.Operation]framework.OperationHandler{
					logical.ListOperation:   NewOperationHandler(b.gcListDatabases, propsListDatabases),
					logical.ReadOperation:   NewOperationHandler(b.gcGetCluster, propsGcGetCluster),
					logical.DeleteOperation: NewOperationHandler(b.gcPurgeCluster, propsGcPurgeCluster),
				},
				HelpSynopsis:    helpSynopsisGCClusterOps,
				HelpDescription: helpDescriptionGCClusterOps,
			},
			{
				Pattern: "gc/cluster/" + framework.GenericNameRegex("cluster") + "/" + framework.GenericNameRegex("database"),
				Fields: map[string]*framework.FieldSchema{
					"cluster": {
						Type:        framework.TypeString,
						Description: "Name of the database cluster",
					},
					"database": {
						Type:        framework.TypeString,
						Description: "Name of the database",
					},
				},
				Operations: map[logical.Operation]framework.OperationHandler{
					logical.ReadOperation:   NewOperationHandler(b.gcGetDatabase, propsGcGetDatabase),
					logical.DeleteOperation: NewOperationHandler(b.gcPurgeDatabase, propsGcPurgeDatabase),
				},
				HelpSynopsis:    helpSynopsisGCDbOps,
				HelpDescription: helpDescriptionGCDbOps,
			},
		},
		Help: helpDescriptionBackend,
	}

	return &b
}

type connType int

func (c connType) String() string {
	switch c {
	case connTypeRoot:
		return "root"
	case connTypeMgmt:
		return "management"
	default:
		return "unknown"
	}
}

const (
	connTypeRoot connType = iota
	connTypeMgmt
)

func (b *backend) getConn(ctx context.Context, storage logical.Storage, connT connType, cluster, db string) (*sql.DB, error) {
	entry, err := storage.Get(ctx, PathCluster.For(cluster))
	if err != nil {
		return nil, err
	}

	if entry == nil {
		return nil, fmt.Errorf("configuration for %s cluster does not exist", cluster)
	}

	c := &ClusterConfig{}
	err = entry.DecodeJSON(c)
	if err != nil {
		return nil, err
	}

	var dsn string
	if db != "" {
		dsn = c.dsnForDb(connT, db)
	} else {
		dsn = c.dsn(connT)
	}

	conn, err := b.makeConn(dsn)
	if err != nil {
		return nil, err
	}

	conn.SetConnMaxLifetime(time.Duration(c.MaxConnectionLifetime) * time.Second)
	conn.SetMaxIdleConns(c.MaxIdleConnections)
	conn.SetMaxOpenConns(c.MaxOpenConnections)

	return conn, nil
}
