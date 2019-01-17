package backend

import (
	"context"
	"fmt"
	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
)

func (b *backend) pathCloneUpdate(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	targetHost := data.Get("host").(string)
	if targetHost == "" {
		return logical.ErrorResponse(fmt.Sprintf("Invalid host name %s", targetHost)), nil
	}

	targetPort := data.Get("port").(int)
	if targetPort <1 || targetPort > 65535 {
		return logical.ErrorResponse(fmt.Sprintf("Invalid port number %d, a valid port number between 1 and 65535 is required", targetPort)), nil
	}

	inheritDeleted := data.Get("inherit_deleted_db").(bool)

	clusterName := data.Get("cluster").(string)
	cluster, err := loadClusterEntry(ctx, req.Storage, clusterName)
	if err == ErrNotFound {
		return logical.ErrorResponse(fmt.Sprintf("Source cluster %s is not configured. Use cluster/%s to configure it first", clusterName, clusterName)), nil
	}

	if err != nil {
		return nil, err
	}

	if cluster.IsDisabled() {
		return logical.ErrorResponse(fmt.Sprintf("Deleted source cluster %s cannot be cloned", clusterName)), nil
	}

	targetName := data.Get("target").(string)
	if targetName == "" {
		return logical.ErrorResponse("target cluster name cannot be empty"), nil
	}

	existing, err := loadClusterEntry(ctx, req.Storage, targetName)
	if err != nil && err != ErrNotFound {
		return nil, err
	}

	if existing != nil {
		return logical.ErrorResponse(fmt.Sprintf("Duplicate value for target cluster %s. A cluster with name %s is already configured", targetName, targetName)), nil
	}

	cluster.Host = targetHost
	cluster.Port = targetPort
	resp := &logical.Response{}

	db, err := b.makeConn(cluster.dsn(connTypeMgmt))
	if err != nil {
		return nil, fmt.Errorf("failed to connect with clone as existing management user. error: %s", err)
	}
	err = db.Close()
	if err != nil {
		resp.AddWarning(fmt.Sprintf("failed to close old management user connection. %s", err))
	}

	db, err = b.makeConn(cluster.dsn(connTypeRoot))
	if err != nil {
		return nil, fmt.Errorf("failed to connect with clone as existing root user. error: %s", err)
	}

	newMgmtPass, err := updatePassword(ctx, db, cluster.ManagementRole)
	if err != nil {
		return nil, fmt.Errorf("failed to rotate the password for management user. %s", err)
	}
	cluster.ManagementPassword = newMgmtPass

	newRootPass, err := updatePassword(ctx, db, cluster.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to rotate the password for root user. %s", err)
	}
	cluster.Password = newRootPass

	err = storeClusterEntry(ctx, req.Storage, targetName, cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to store the configuration for clone cluster. %s", err)
	}

	total, success, errs := cloneDbConfig(ctx, req.Storage, clusterName, targetName, inheritDeleted)
	if errs != nil && len(errs) > 0 {
		for _, e := range errs {
			resp.AddWarning(e.Error())
		}
	}

	resp.AddWarning(fmt.Sprintf("%d of %d databases inherited successfully", success, total))
	return resp, nil
}

func cloneDbConfig(ctx context.Context, storage logical.Storage, source, target string, inheritDeleted bool) (total, success int, errs []error) {
	dbs, err := storage.List(ctx, PathDatabase.For(source, ""))
	if err != nil {
		return 0, 0, []error{
			fmt.Errorf("failed to list databases in existing cluster. %s", err),
		}
	}

	total = len(dbs)
	for _, dbname := range dbs {
		db, err := loadDbEntry(ctx, storage, source, dbname)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to load configuration for existing database %s. %s", dbname, err))
			continue
		}

		if db.IsDisabled() && !inheritDeleted {
			continue
		}

		db.Cluster = target
		err = storeDbEntry(ctx, storage, target, dbname, db)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to store configuration for cloned database %s. %s", dbname, err))
			continue
		}

		success += 1
	}

	return
}
