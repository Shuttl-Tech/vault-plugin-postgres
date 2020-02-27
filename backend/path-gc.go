package backend

import (
	"context"
	"fmt"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

func listDisabledClusters(ctx context.Context, storage logical.Storage) ([]string, error) {
	clusters, err := storage.List(ctx, PathCluster.For(""))
	if err != nil {
		return nil, err
	}

	var results []string
	for _, c := range clusters {
		cluster, err := loadClusterEntry(ctx, storage, c)
		if err != nil {
			return nil, err
		}

		if cluster.IsDisabled() {
			results = append(results, c)
		}
	}

	return results, nil
}

func (b *backend) gcListClusters(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	results, err := listDisabledClusters(ctx, req.Storage)
	if err != nil {
		return nil, err
	}

	return logical.ListResponse(results), nil
}

func (b *backend) gcGetCluster(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	clusterName := data.Get("cluster").(string)
	c, err := loadClusterEntry(ctx, req.Storage, clusterName)
	if err == ErrNotFound {
		return logical.ErrorResponse(fmt.Sprintf("Cluster with name %s is not registered", clusterName)), nil
	}

	if err != nil {
		return nil, err
	}

	if !c.IsDisabled() {
		return logical.ErrorResponse(fmt.Sprintf("Cluster %s is not marked for GC. Delete the cluster before invoking GC operations on it", clusterName)), nil
	}

	return &logical.Response{
		Data: c.AsMap(),
	}, nil
}

func listDisabledDatabases(ctx context.Context, cluster string, storage logical.Storage) ([]string, error) {
	entries, err := storage.List(ctx, PathDatabase.For(cluster, ""))
	if err != nil {
		return nil, err
	}

	var results []string
	for _, dbname := range entries {
		db, err := loadDbEntry(ctx, storage, cluster, dbname)
		if err != nil {
			return nil, err
		}

		if db.IsDisabled() {
			results = append(results, dbname)
		}
	}

	return results, nil
}

func (b *backend) gcListDatabases(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	cluster := data.Get("cluster").(string)
	databases, err := listDisabledDatabases(ctx, cluster, req.Storage)
	if err != nil {
		return nil, err
	}

	return logical.ListResponse(databases), nil
}

func (b *backend) gcPurgeCluster(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	cn := data.Get("cluster").(string)
	cluster, err := loadClusterEntry(ctx, req.Storage, cn)
	if err != nil {
		return nil, err
	}

	if !cluster.IsDisabled() {
		return logical.ErrorResponse(fmt.Sprintf("Cluster %s is not marked for GC. Delete the cluster using cluster/:name endpoint before invoking GC operation on it", cn)), nil
	}

	err = req.Storage.Delete(ctx, PathCluster.For(cn))
	if err != nil {
		return nil, err
	}

	return &logical.Response{
		Data: map[string]interface{}{},
	}, nil
}

func (b *backend) gcGetDatabase(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	cn := data.Get("cluster").(string)
	dn := data.Get("database").(string)

	dbC, err := loadDbEntry(ctx, req.Storage, cn, dn)
	if err == ErrNotFound {
		return logical.ErrorResponse(fmt.Sprintf("Database %s does not exist in cluster %s", dn, cn)), nil
	}

	if err != nil {
		return nil, err
	}

	if !dbC.IsDisabled() {
		return logical.ErrorResponse(fmt.Sprintf("Database %s is not marked for GC. Delete the database before invoking GC operations on it", dn)), nil
	}

	return &logical.Response{
		Data: dbC.AsMap(),
	}, nil
}

func (b *backend) gcPurgeDatabase(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	cn := data.Get("cluster").(string)
	dn := data.Get("database").(string)

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

	if !dbC.IsDisabled() {
		return logical.ErrorResponse(fmt.Sprintf("Database %s is not marked for GC. Delete the databaes from cluster/:cluster/:database endpoint before invoking GC operation on it", dn)), nil
	}

	err = req.Storage.Delete(ctx, PathDatabase.For(cn, dn))
	if err != nil {
		return nil, err
	}

	return &logical.Response{}, nil
}
