package backend

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/vault/helper/dbtxn"
	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
	"github.com/lib/pq"
)

type ClusterConfig struct {
	Host                  string `json:"host" mapstructure:"host"`
	Port                  int    `json:"port" mapstructure:"port"`
	Username              string `json:"username" mapstructure:"username"`
	Password              string `json:"password" mapstructure:"password"`
	ManagementRole        string `json:"management_role" mapstructure:"management_role"`
	ManagementPassword    string `json:"management_password" mapstructure:"management_password"`
	MaxOpenConnections    int    `json:"max_open_connections" mapstructure:"max_open_connections"`
	MaxIdleConnections    int    `json:"max_idle_connections" mapstructure:"max_idle_connections"`
	MaxConnectionLifetime int    `json:"max_connection_lifetime" mapstructure:"max_connection_lifetime"`
	Database              string `json:"database" mapstructure:"database"`
	Disabled              *bool  `json:"disabled" mapstructure:"disabled"`
	SSLMode               string `json:"ssl_mode" mapstructure:"ssl_mode"`
}

func (c *ClusterConfig) AsMap() map[string]interface{} {
	return map[string]interface{}{
		"host":                    c.Host,
		"port":                    c.Port,
		"username":                c.Username,
		"password":                c.Password,
		"max_open_connections":    c.MaxOpenConnections,
		"max_idle_connections":    c.MaxIdleConnections,
		"max_connection_lifetime": c.MaxConnectionLifetime,
		"database":                c.Database,
		"disabled":                c.IsDisabled(),
		"ssl_mode":                c.SSLMode,
		"management_role":         c.ManagementRole,
		"management_password":     c.ManagementPassword,
	}
}

func (c *ClusterConfig) IsDisabled() bool {
	if c.Disabled == nil {
		return false
	}

	return *c.Disabled
}

func (c *ClusterConfig) Disable() {
	d := true
	c.Disabled = &d
}

func (c *ClusterConfig) validate() error {
	if c.Host == "" {
		return fmt.Errorf("Invalid host value")
	}

	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("Invalid port number %d", c.Port)
	}

	if c.Username == "" {
		return fmt.Errorf("Username must be set")
	}

	if c.Database == "" {
		return fmt.Errorf("Maintenance database must be set")
	}

	switch c.SSLMode {
	case "disable", "require", "verify-ca", "verify-full":
	default:
		return fmt.Errorf("Invalid ssl_mode %s, valid options are 'disable', 'require', 'verify-ca', or 'verify-full'", c.SSLMode)
	}

	return nil
}

func (c *ClusterConfig) dsn(t connType) string {
	return c.dsnForDb(t, c.Database)
}

func (c *ClusterConfig) dsnForDb(t connType, db string) string {
	u, p := c.Username, c.Password
	if t == connTypeMgmt {
		u, p = c.ManagementRole, c.ManagementPassword
	}

	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?timezone=utc&sslmode=%s",
		u, p, c.Host, c.Port, db, c.SSLMode)
}

func (c *ClusterConfig) loadFromFields(data *framework.FieldData) error {
	for k := range data.Schema {
		switch k {
		case "host":
			c.Host = data.Get("host").(string)
		case "port":
			c.Port = data.Get("port").(int)
		case "username":
			c.Username = data.Get("username").(string)
		case "password":
			c.Password = data.Get("password").(string)
		case "max_open_connections":
			c.MaxOpenConnections = data.Get("max_open_connections").(int)
		case "max_idle_connections":
			c.MaxIdleConnections = data.Get("max_idle_connections").(int)
		case "max_connection_lifetime":
			c.MaxConnectionLifetime = data.Get("max_connection_lifetime").(int)
		case "database":
			c.Database = data.Get("database").(string)
		case "ssl_mode":
			c.SSLMode = data.Get("ssl_mode").(string)
		}
	}

	return c.validate()
}

func loadClusterEntry(ctx context.Context, storage logical.Storage, name string) (*ClusterConfig, error) {
	entry, err := storage.Get(ctx, PathCluster.For(name))
	if err != nil {
		return nil, err
	}

	if entry == nil {
		return nil, ErrNotFound
	}

	c := &ClusterConfig{}
	err = entry.DecodeJSON(c)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func storeClusterEntry(ctx context.Context, storage logical.Storage, name string, cfg *ClusterConfig) error {
	entry, err := logical.StorageEntryJSON(PathCluster.For(name), cfg)
	if err != nil {
		return err
	}

	return storage.Put(ctx, entry)
}

func updatePassword(ctx context.Context, db *sql.DB, username string) (string, error) {
	newPass, err := uuid.GenerateUUID()
	if err != nil {
		return "", err
	}

	cpQ := map[string]string{
		"user":     pq.QuoteIdentifier(username),
		"password": newPass,
	}

	err = dbtxn.ExecuteDBQuery(ctx, db, cpQ, queryUpdatePassword)
	if err != nil {
		return "", err
	}

	return newPass, nil
}

func createManagementRole(ctx context.Context, db *sql.DB) (string, string, error) {
	mgmtRoleName, err := uuid.GenerateUUID()
	if err != nil {
		return "", "", err
	}

	roleName := fmt.Sprintf("v-manage-%s", mgmtRoleName)
	if len(roleName) > 63 {
		roleName = roleName[:63]
	}

	rolePass, err := uuid.GenerateUUID()
	if err != nil {
		return "", "", err
	}

	crQ := map[string]string{
		"user":     pq.QuoteIdentifier(roleName),
		"password": rolePass,
	}

	err = dbtxn.ExecuteDBQuery(ctx, db, crQ, queryCreateManagementRole)
	if err != nil {
		return "", "", err
	}

	return roleName, rolePass, nil
}

func (b *backend) pathClusterRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	clusterName := data.Get("cluster").(string)
	c, err := loadClusterEntry(ctx, req.Storage, clusterName)
	if err == ErrNotFound {
		return logical.ErrorResponse(fmt.Sprintf("Cluster with name %s is not registered", clusterName)), nil
	}

	if err != nil {
		return nil, err
	}

	if c.IsDisabled() {
		return logical.ErrorResponse(fmt.Sprintf("Cluster %s is deleted. Use gc/cluster to manage deleted clusters", clusterName)), nil
	}

	return &logical.Response{
		Data: c.AsMap(),
	}, nil
}

func (b *backend) pathClusterUpdate(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	clusterName := data.Get("cluster").(string)
	configName := PathCluster.For(clusterName)
	existing, err := loadClusterEntry(ctx, req.Storage, clusterName)
	if err != nil && err != ErrNotFound {
		return nil, err
	}

	if existing != nil && existing.IsDisabled() {
		return logical.ErrorResponse(fmt.Sprintf("Cluster %s is deleted. Use gc/cluster to manage deleted clusters", clusterName)), nil
	}

	c := &ClusterConfig{}
	err = c.loadFromFields(data)
	if err != nil {
		return logical.ErrorResponse(err.Error()), nil
	}

	db, err := b.makeConn(c.dsn(connTypeRoot))
	if err != nil {
		return logical.ErrorResponse(err.Error()), nil
	}
	defer func() {
		_ = db.Close()
	}()

	mgmtRole, mgmtPass, err := createManagementRole(ctx, db)
	if err != nil {
		return nil, err
	}

	c.ManagementRole = mgmtRole
	c.ManagementPassword = mgmtPass

	newPass, err := updatePassword(ctx, db, c.Username)
	if err != nil {
		return nil, err
	}

	c.Password = newPass

	err = storeClusterEntry(ctx, req.Storage, clusterName, c)
	if err != nil {
		return nil, err
	}

	resp := &logical.Response{}
	resp.AddWarning("The password has been changed by Vault. Old password will no longer work")
	resp.AddWarning(fmt.Sprintf("A management role with name '%s' has been created by Vault", mgmtRole))

	err = b.flushConn(configName)
	if err != nil {
		resp.AddWarning("Failed to flush existing connections after update. error: " + err.Error())
	}

	return resp, nil
}

func (b *backend) pathClusterDelete(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	clusterName := data.Get("cluster").(string)
	configName := PathCluster.For(clusterName)

	c, err := loadClusterEntry(ctx, req.Storage, clusterName)
	if err == ErrNotFound {
		return logical.ErrorResponse(fmt.Sprintf("Cluster with name %s is not registered", clusterName)), nil
	}

	if err != nil {
		return nil, err
	}

	if c.IsDisabled() {
		return logical.ErrorResponse(fmt.Sprintf("Cluster %s is deleted. Use gc/cluster to manage deleted clusters", clusterName)), nil
	}

	c.Disable()

	err = storeClusterEntry(ctx, req.Storage, clusterName, c)
	if err != nil {
		return nil, err
	}

	warnings := []string{
		"Use gc/cluster to manage deleted clusters",
	}

	err = b.flushAllConn(configName)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("Failed to revoke active connections. error: %s", err))
	}

	return &logical.Response{
		Warnings: warnings,
	}, nil
}

func (b *backend) pathClustersList(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	entries, err := req.Storage.List(ctx, PathCluster.For(""))
	if err != nil {
		return nil, err
	}

	return logical.ListResponse(entries), nil
}

func (b *backend) makeConn(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("Error validating connection. Error: %s", err)
	}

	if err = db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("Error validating connection. Error: %s", err)
	}

	return db, nil
}
