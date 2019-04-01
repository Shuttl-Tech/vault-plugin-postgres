package backend

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
	"strings"
	"sync"
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

	db map[string]*sql.DB
	sync.Mutex
}

func Factory(ctx context.Context, c *logical.BackendConfig) (logical.Backend, error) {
	b := New(c)
	if err := b.Setup(ctx, c); err != nil {
		return nil, err
	}
	return b, nil
}

func New(c *logical.BackendConfig) *backend {
	b := backend{
		db: map[string]*sql.DB{},
	}

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
				Callbacks: map[logical.Operation]framework.OperationFunc{
					logical.ReadOperation: b.pathInfo,
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
				Callbacks: map[logical.Operation]framework.OperationFunc{
					logical.UpdateOperation: b.pathMetadataUpdate,
					logical.ReadOperation:   b.pathMetadataRead,
					logical.ListOperation:   b.pathMetadataList,
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
				Callbacks: map[logical.Operation]framework.OperationFunc{
					logical.DeleteOperation: b.pathMetadataDelete,
				},
				HelpSynopsis: "Delete metadata using ID",
			},
			{
				Pattern: "cluster/?$",
				Callbacks: map[logical.Operation]framework.OperationFunc{
					logical.ListOperation: b.pathClustersList,
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
				Callbacks: map[logical.Operation]framework.OperationFunc{
					logical.ReadOperation:   b.pathClusterRead,
					logical.UpdateOperation: b.pathClusterUpdate,
					logical.DeleteOperation: b.pathClusterDelete,
					logical.ListOperation:   b.pathDatabasesList,
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
				Callbacks: map[logical.Operation]framework.OperationFunc{
					logical.UpdateOperation: b.pathCloneUpdate,
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
				Callbacks: map[logical.Operation]framework.OperationFunc{
					logical.UpdateOperation: b.pathDatabaseUpdate,
					logical.ReadOperation:   b.pathDatabaseRead,
					logical.DeleteOperation: b.pathDatabaseDelete,
				},
				HelpSynopsis:    helpSynopsisDatabase,
				HelpDescription: helpDescriptionDatabase,
			},
			{
				Pattern: "roles/?$",
				Callbacks: map[logical.Operation]framework.OperationFunc{
					logical.ListOperation: b.pathRoleList,
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
				Callbacks: map[logical.Operation]framework.OperationFunc{
					logical.UpdateOperation: b.pathRoleUpdate,
					logical.ReadOperation:   b.pathRoleRead,
					logical.DeleteOperation: b.pathRoleDelete,
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
				Callbacks: map[logical.Operation]framework.OperationFunc{
					logical.ReadOperation: b.secretCredsCreate,
				},
				HelpSynopsis:    helpSynopsisCreds,
				HelpDescription: helpDescriptionCreds,
			},
		},
		Help: helpDescriptionBackend,
	}

	return &b
}

func (b *backend) flushConn(name string) error {
	b.Lock()
	defer b.Unlock()

	conn, ok := b.db[name]
	if !ok {
		return nil
	}

	err := conn.Close()
	delete(b.db, name)
	return err
}

// flush all connections for a cluster.
// this function will close all active connections on
// matching root or management connection prefix and delete
// the entries from connection pool
func (b *backend) flushAllConn(name string) error {
	rootConnPrefix := fmt.Sprintf("%s/%s", connTypeRoot.String(), name)
	mgmtConnPrefix := fmt.Sprintf("%s/%s", connTypeMgmt.String(), name)

	var (
		d, e []string
	)

	b.Lock()
	defer b.Unlock()

	for k, conn := range b.db {
		if strings.HasPrefix(k, rootConnPrefix) || strings.HasPrefix(k, mgmtConnPrefix) {
			d = append(d, k)
			if err := conn.Close(); err != nil {
				e = append(e, err.Error())
			}
		}
	}

	for _, k := range d {
		delete(b.db, k)
	}

	if len(e) > 0 {
		return fmt.Errorf("%s", strings.Join(e, "\n"))
	}

	return nil
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
	var confPath string
	if db == "" {
		confPath = PathDatabase.For(cluster, db)
	} else {
		confPath = PathCluster.For(cluster)
	}

	confPath = fmt.Sprintf("%s/%s", connT.String(), confPath)

	b.Lock()
	conn, ok := b.db[confPath]
	b.Unlock()

	if ok {
		return conn, nil
	}

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

	conn, err = b.makeConn(dsn)
	if err != nil {
		return nil, err
	}

	conn.SetConnMaxLifetime(time.Duration(c.MaxConnectionLifetime) * time.Second)
	conn.SetMaxIdleConns(c.MaxIdleConnections)
	conn.SetMaxOpenConns(c.MaxOpenConnections)

	b.Lock()
	b.db[confPath] = conn
	b.Unlock()

	return conn, nil
}
