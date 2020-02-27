package backend

import (
	"context"
	"fmt"
	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/helper/dbtxn"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/lib/pq"
)

type DbConfig struct {
	Cluster      string `json:"cluster" mapstructure:"cluster"`
	Database     string `json:"database" mapstructure:"database"`
	ObjectsOwner string `json:"objects_owner" mapstructure:"objects_owner"`
	Disabled     *bool  `json:"disabled" mapstructure:"disabled"`
}

func (db *DbConfig) AsMap() map[string]interface{} {
	return map[string]interface{}{
		"cluster":       db.Cluster,
		"database":      db.Database,
		"disabled":      db.IsDisabled(),
		"objects_owner": db.ObjectsOwner,
	}
}

func (db *DbConfig) IsDisabled() bool {
	if db.Disabled == nil {
		return false
	}

	return *db.Disabled
}

func (db *DbConfig) Disable() {
	d := true
	db.Disabled = &d
}

func (db *DbConfig) validate() error {
	if db.Database == "" {
		return fmt.Errorf("Database name is not set")
	}

	if db.Cluster == "" {
		return fmt.Errorf("Cluster name is not set")
	}

	if db.ObjectsOwner == "" {
		return fmt.Errorf("Objects owner is not set")
	}

	return nil
}

func loadDbEntry(ctx context.Context, storage logical.Storage, cluster, db string) (*DbConfig, error) {
	entry, err := storage.Get(ctx, PathDatabase.For(cluster, db))
	if err != nil {
		return nil, err
	}

	if entry == nil {
		return nil, ErrNotFound
	}

	d := &DbConfig{}
	err = entry.DecodeJSON(d)
	if err != nil {
		return nil, err
	}

	return d, nil
}

func (b *backend) pathDatabaseUpdate(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	cn := data.Get("cluster").(string)
	dn := data.Get("database").(string)

	c, err := loadClusterEntry(ctx, req.Storage, cn)
	if err == ErrNotFound {
		return logical.ErrorResponse(fmt.Sprintf("Cluster with name %s is not registered", cn)), nil
	}

	if err != nil {
		return nil, err
	}

	if c.IsDisabled() {
		return logical.ErrorResponse(fmt.Sprintf("Cluster %s is deleted. Cannot register new databases in deleted cluster", cn)), nil
	}

	dbExisting, err := loadDbEntry(ctx, req.Storage, cn, dn)
	if err != ErrNotFound && err != nil {
		return nil, err
	}

	if dbExisting != nil {
		return logical.ErrorResponse(fmt.Sprintf("Database %s is already registered in cluster %s", dn, cn)), nil
	}

	objectsOwner := data.Get("objects_owner_role").(string)
	if objectsOwner == "" {
		objOwnId, err := uuid.GenerateUUID()
		if err != nil {
			return nil, err
		}

		objectsOwner = fmt.Sprintf("v-objown-%s-%s", dn, objOwnId)
	}

	if len(objectsOwner) > 63 {
		objectsOwner = objectsOwner[:63]
	}

	if data.Get("initialize").(bool) {
		createNewDb := data.Get("create_db").(bool)
		if err = initializeDb(ctx, req.Storage, b, c, cn, dn, objectsOwner, createNewDb); err != nil {
			return nil, err
		}
	}

	dbC := &DbConfig{
		Cluster:      cn,
		Database:     dn,
		ObjectsOwner: objectsOwner,
	}

	err = storeDbEntry(ctx, req.Storage, cn, dn, dbC)
	if err != nil {
		return nil, err
	}

	return &logical.Response{}, nil
}

func initializeDb(ctx context.Context, storage logical.Storage, b *backend, c *ClusterConfig, cn, dn, objectsOwner string, createNewDb bool) error {
	clusterConn, err := b.getConn(ctx, storage, connTypeRoot, cn, c.Database)
	if err != nil {
		return err
	}

	dbQV := map[string]string{
		"database": pq.QuoteIdentifier(dn),
	}

	if createNewDb {
		err = dbtxn.ExecuteDBQuery(ctx, clusterConn, dbQV, queryCreateDb)
		if err != nil {
			return err
		}
	}

	dbConn, err := b.getConn(ctx, storage, connTypeMgmt, cn, dn)
	if err != nil {
		return err
	}

	rQV := map[string]string{
		"role_name":             pq.QuoteIdentifier(objectsOwner),
		"role_group_management": pq.QuoteIdentifier(c.ManagementRole),
		"role_group_root":       pq.QuoteIdentifier(c.Username),
	}

	tx, err := dbConn.Begin()
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	qSetupRole := []string{queryCreateObjectsOwnerRole, queryGrantAll}
	for _, q := range qSetupRole {
		if err = dbtxn.ExecuteTxQuery(ctx, tx, rQV, q); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func storeDbEntry(ctx context.Context, storage logical.Storage, clusterName, dbName string, db *DbConfig) error {
	dEntry, err := logical.StorageEntryJSON(PathDatabase.For(clusterName, dbName), db)
	if err != nil {
		return err
	}

	return storage.Put(ctx, dEntry)
}

func (b *backend) pathDatabaseDelete(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	cn := data.Get("cluster").(string)
	dn := data.Get("database").(string)

	cEntry, err := req.Storage.Get(ctx, PathCluster.For(cn))
	if err != nil {
		return nil, err
	}

	if cEntry == nil {
		return logical.ErrorResponse(fmt.Sprintf("Cluster with name %s is not registered", cn)), nil
	}

	c := &ClusterConfig{}
	err = cEntry.DecodeJSON(c)
	if err != nil {
		return nil, err
	}

	if c.IsDisabled() {
		return logical.ErrorResponse(fmt.Sprintf("Cluster %s is deleted. Database %s is automatically marked as deleted", cn, dn)), nil
	}

	dEntry, err := req.Storage.Get(ctx, PathDatabase.For(cn, dn))
	if err != nil {
		return nil, err
	}

	if dEntry == nil {
		return logical.ErrorResponse(fmt.Sprintf("Database %s does not exist in cluster %s", dn, cn)), nil
	}

	dbC := &DbConfig{}
	err = dEntry.DecodeJSON(&dbC)
	if err != nil {
		return nil, err
	}

	if dbC.IsDisabled() {
		return logical.ErrorResponse(fmt.Sprintf("Database %s is already deleted", dn)), nil
	}

	dbC.Disable()
	dEntry, err = logical.StorageEntryJSON(PathDatabase.For(cn, dn), dbC)
	if err != nil {
		return nil, err
	}

	err = req.Storage.Put(ctx, dEntry)
	if err != nil {
		return nil, err
	}

	return &logical.Response{}, nil
}

func (b *backend) pathDatabaseRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	cn := data.Get("cluster").(string)
	dn := data.Get("database").(string)

	cEntry, err := req.Storage.Get(ctx, PathCluster.For(cn))
	if err != nil {
		return nil, err
	}

	if cEntry == nil {
		return logical.ErrorResponse(fmt.Sprintf("Cluster with name %s is not registered", cn)), nil
	}

	c := &ClusterConfig{}
	err = cEntry.DecodeJSON(c)
	if err != nil {
		return nil, err
	}

	if c.IsDisabled() {
		return logical.ErrorResponse(fmt.Sprintf("Database %s in deleted cluster %s is marked as deleted", dn, cn)), nil
	}

	dbC, err := loadDbEntry(ctx, req.Storage, cn, dn)
	if err == ErrNotFound {
		return logical.ErrorResponse(fmt.Sprintf("Database %s does not exist in cluster %s", dn, cn)), nil
	}

	if err != nil {
		return nil, err
	}

	if dbC.IsDisabled() {
		return logical.ErrorResponse(fmt.Sprintf("Database %s is deleted. Use gc/ to manage deleted databases", dn)), nil
	}

	return &logical.Response{
		Data: dbC.AsMap(),
	}, nil
}

func (b *backend) pathDatabasesList(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	cluster := data.Get("cluster").(string)
	entries, err := req.Storage.List(ctx, PathDatabase.For(cluster, ""))
	if err != nil {
		return nil, err
	}

	return logical.ListResponse(entries), nil
}
