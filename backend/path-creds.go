package backend

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/vault/helper/dbtxn"
	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
	"github.com/lib/pq"
)

var defaultCreationSQL = []string{
	"create role {{user}} with login password '{{password}}' inherit in role {{objects_owner}} valid until '{{expiration}}' role {{group}}",
	"alter default privileges for role {{user}} grant all privileges on tables to {{objects_owner}}",
	"alter default privileges for role {{user}} grant all privileges on sequences to {{objects_owner}}",
}

var defaultRevocationSQL = []string{
	"set role {{user}}",
	"reassign owned by {{user}} to {{objects_owner}}",
	"drop owned by {{user}}",
	"reset role",
	"revoke {{objects_owner}} from {{user}}",
	"revoke connect on database {{database}} from {{user}}",
	"drop role if exists {{user}}",
}

func (b *backend) secretCredsCreate(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	clusterName := data.Get("cluster").(string)
	databaseName := data.Get("database").(string)
	roleName := data.Get("role").(string)

	cluster, err := loadClusterEntry(ctx, req.Storage, clusterName)
	if err == ErrNotFound {
		return logical.ErrorResponse(fmt.Sprintf("Cluster %s is not configured", clusterName)), nil
	}

	if err != nil {
		return nil, err
	}

	if cluster.IsDisabled() {
		return logical.ErrorResponse(fmt.Sprintf("Cluster %s is marked as deleted. Cannot generate new credentials", clusterName)), nil
	}

	database, err := loadDbEntry(ctx, req.Storage, clusterName, databaseName)
	if err == ErrNotFound {
		return logical.ErrorResponse(fmt.Sprintf("Database %s is not configured", databaseName)), nil
	}

	if err != nil {
		return nil, err
	}

	if database.IsDisabled() {
		return logical.ErrorResponse(fmt.Sprintf("Database %s is marked as deleted. Cannot generate new credentials", databaseName)), nil
	}

	role, err := loadRoleEntry(ctx, req.Storage, roleName)
	if err == ErrNotFound {
		return logical.ErrorResponse(fmt.Sprintf("Role %s is not configured", roleName)), nil
	}

	displayName := req.DisplayName
	if len(displayName) > 26 {
		displayName = displayName[:26]
	}
	userUUID, err := uuid.GenerateUUID()
	if err != nil {
		return nil, err
	}
	username := fmt.Sprintf("%s-%s", displayName, userUUID)
	if len(username) > 63 {
		username = username[:63]
	}

	password, err := uuid.GenerateUUID()
	if err != nil {
		return nil, err
	}

	ttl, _, err := framework.CalculateTTL(b.System(), 0, role.GetDefaultTTL(), 0, role.GetDefaultTTL(), 0, time.Time{})
	if err != nil {
		return nil, err
	}

	expiration := time.Now().Add(ttl).Format("2006-01-02 15:04:05-0700")

	db, err := b.getConn(ctx, req.Storage, connTypeMgmt, clusterName, databaseName)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = db.Close()
	}()

	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	m := map[string]string{
		"user":          pq.QuoteIdentifier(username),
		"password":      password,
		"expiration":    expiration,
		"objects_owner": pq.QuoteIdentifier(database.ObjectsOwner),
		"group":         pq.QuoteIdentifier(cluster.ManagementRole),
	}

	for _, query := range role.CreationStatement {
		query = strings.TrimSpace(query)
		if len(query) == 0 {
			continue
		}

		if err := dbtxn.ExecuteTxQuery(ctx, tx, m, query); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	sec := map[string]interface{}{
		"username": username,
		"password": password,
	}

	internalSec := map[string]interface{}{
		"role":     roleName,
		"username": username,
		"cluster":  clusterName,
		"database": databaseName,
	}

	resp := b.Secret(SecretCredsType).Response(sec, internalSec)
	resp.Secret.TTL = role.GetDefaultTTL()
	resp.Secret.MaxTTL = role.GetMaxTTL()
	return resp, nil
}

func (b *backend) secretCredsRenew(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	roleName, err := getInternalStr("role", req.Secret.InternalData)
	if err != nil {
		return nil, err
	}

	username, err := getInternalStr("username", req.Secret.InternalData)
	if err != nil {
		return nil, err
	}

	clusterName, err := getInternalStr("cluster", req.Secret.InternalData)
	if err != nil {
		return nil, err
	}

	databaseName, err := getInternalStr("database", req.Secret.InternalData)
	if err != nil {
		return nil, err
	}

	role, err := loadRoleEntry(ctx, req.Storage, roleName)
	if err == ErrNotFound {
		return logical.ErrorResponse(fmt.Sprintf("Role %s is not configured", roleName)), nil
	}

	if err != nil {
		return nil, err
	}

	cluster, err := loadClusterEntry(ctx, req.Storage, clusterName)
	if err == ErrNotFound {
		return logical.ErrorResponse(fmt.Sprintf("Configuration for cluster %s is not available", clusterName)), nil
	}

	if err != nil {
		return nil, err
	}

	if cluster.IsDisabled() {
		return logical.ErrorResponse(fmt.Sprintf("Cluster %s is marked as deleted. Cannot renew credentials", clusterName)), nil
	}

	database, err := loadDbEntry(ctx, req.Storage, clusterName, databaseName)
	if err == ErrNotFound {
		return logical.ErrorResponse(fmt.Sprintf("Configuration for database %s is not available", databaseName)), nil
	}

	if err != nil {
		return nil, err
	}

	if database.IsDisabled() {
		return logical.ErrorResponse(fmt.Sprintf("Database %s is marked as deleted. Cannot renew credentials", databaseName)), nil
	}

	ttl, warnings, err := framework.CalculateTTL(b.System(), req.Secret.Increment, role.GetDefaultTTL(), 0, role.GetMaxTTL(), role.GetMaxTTL(), req.Secret.IssueTime)
	if ttl > 0 {
		expiration := time.Now().UTC().Add(ttl).Add(5 * time.Second).Format("2006-01-02 15:04:05+00")
		m := map[string]string{
			"user":       pq.QuoteIdentifier(username),
			"expiration": expiration,
		}

		db, err := b.getConn(ctx, req.Storage, connTypeMgmt, clusterName, databaseName)
		if err != nil {
			return nil, err
		}
		defer func() {
			_ = db.Close()
		}()

		err = dbtxn.ExecuteDBQuery(ctx, db, m, queryRenewExpiry)
		if err != nil {
			return nil, err
		}
	}

	resp := &logical.Response{
		Secret:   req.Secret,
		Warnings: warnings,
	}

	resp.Secret.TTL = ttl
	resp.Secret.MaxTTL = role.GetMaxTTL()
	return resp, nil
}

func (b *backend) secretCredsRevoke(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	roleName, err := getInternalStr("role", req.Secret.InternalData)
	if err != nil {
		return nil, err
	}

	username, err := getInternalStr("username", req.Secret.InternalData)
	if err != nil {
		return nil, err
	}

	clusterName, err := getInternalStr("cluster", req.Secret.InternalData)
	if err != nil {
		return nil, err
	}

	databaseName, err := getInternalStr("database", req.Secret.InternalData)
	if err != nil {
		return nil, err
	}

	resp := &logical.Response{}

	role, err := loadRoleEntry(ctx, req.Storage, roleName)
	if err != ErrNotFound && err != nil {
		return nil, err
	}

	var revocationSQL []string

	if err == ErrNotFound {
		resp.AddWarning(fmt.Sprintf("Role %s was not found, using default revocation SQL", roleName))
		revocationSQL = defaultRevocationSQL
	} else {
		revocationSQL = role.RevocationStatement
	}

	cluster, err := loadClusterEntry(ctx, req.Storage, clusterName)
	if err == ErrNotFound {
		return logical.ErrorResponse(fmt.Sprintf("Configuration for cluster %s cannot be found", clusterName)), nil
	}

	if err != nil {
		return nil, err
	}

	database, err := loadDbEntry(ctx, req.Storage, clusterName, databaseName)
	if err == ErrNotFound {
		return logical.ErrorResponse(fmt.Sprintf("Configuration for database %s cannot be found", databaseName)), nil
	}

	if err != nil {
		return nil, err
	}

	m := map[string]string{
		"user":          pq.QuoteIdentifier(username),
		"database":      pq.QuoteIdentifier(databaseName),
		"objects_owner": pq.QuoteIdentifier(database.ObjectsOwner),
		"group":         pq.QuoteIdentifier(cluster.ManagementRole),
	}

	db, err := b.getConn(ctx, req.Storage, connTypeMgmt, clusterName, databaseName)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = db.Close()
	}()

	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	for idx, query := range revocationSQL {
		query := strings.TrimSpace(query)
		if len(query) == 0 {
			continue
		}

		if err := dbtxn.ExecuteTxQuery(ctx, tx, m, query); err != nil {
			resp.AddWarning(fmt.Sprintf("failed to run revocation query [%d]: %q - %s", idx, query, err))
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return resp, nil
}

func getInternalStr(key string, data map[string]interface{}) (string, error) {
	vr, ok := data[key]
	if !ok {
		return "", fmt.Errorf("secret is missing internal data: %s", key)
	}

	val, ok := vr.(string)
	if !ok {
		return "", fmt.Errorf("raw value for %s internal data is not a string", key)
	}

	return val, nil
}
