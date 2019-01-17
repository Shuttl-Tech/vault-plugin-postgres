package backend

import (
	"context"
	"fmt"
	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
	"time"
)

type RoleConfig struct {
	MaxTTL              int      `json:"max_ttl" mapstructure:"max_ttl"`
	DefaultTTL          int      `json:"default_ttl" mapstructure:"default_ttl"`
	CreationStatement   []string `json:"creation_statement" mapstructure:"creation_statement"`
	RevocationStatement []string `json:"revocation_statement" mapstructure:"revocation_statement"`
}

func (r *RoleConfig) GetDefaultTTL() time.Duration {
	return time.Duration(r.DefaultTTL) * time.Second
}

func (r *RoleConfig) GetMaxTTL() time.Duration {
	return time.Duration(r.MaxTTL) * time.Second
}

func (r *RoleConfig) AsMap() map[string]interface{} {
	return map[string]interface{}{
		"max_ttl":              r.MaxTTL,
		"default_ttl":          r.DefaultTTL,
		"creation_statement":   r.CreationStatement,
		"revocation_statement": r.RevocationStatement,
	}
}

func (r *RoleConfig) loadFromFields(data *framework.FieldData) error {
	for k := range data.Schema {
		switch k {
		case "max_ttl":
			r.MaxTTL = data.Get(k).(int)
		case "default_ttl":
			r.DefaultTTL = data.Get(k).(int)
		case "creation_statement":
			r.CreationStatement = data.Get(k).([]string)
		case "revocation_statement":
			r.RevocationStatement = data.Get(k).([]string)
		}
	}

	return nil
}

func (b *backend) pathRoleUpdate(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	name := data.Get("name").(string)

	c := &RoleConfig{}
	err := c.loadFromFields(data)
	if err != nil {
		return nil, err
	}

	err = storeRoleEntry(ctx, req.Storage, name, c)
	if err != nil {
		return nil, err
	}

	return &logical.Response{}, nil
}

func storeRoleEntry(ctx context.Context, storage logical.Storage, roleName string, role *RoleConfig) error {
	rEntry, err := logical.StorageEntryJSON(PathRole.For(roleName), role)
	if err != nil {
		return err
	}

	err = storage.Put(ctx, rEntry)
	if err != nil {
		return err
	}

	return nil
}

func loadRoleEntry(ctx context.Context, storage logical.Storage, roleName string) (*RoleConfig, error) {
	entry, err := storage.Get(ctx, PathRole.For(roleName))
	if err != nil {
		return nil, err
	}

	if entry == nil {
		return nil, ErrNotFound
	}

	conf := &RoleConfig{}
	err = entry.DecodeJSON(conf)
	if err != nil {
		return nil, err
	}

	return conf, nil
}

func (b *backend) pathRoleRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	name := data.Get("name").(string)
	c, err := loadRoleEntry(ctx, req.Storage, name)
	if err == nil {
		return &logical.Response{
			Data: c.AsMap(),
		}, nil
	}

	if err == ErrNotFound {
		return logical.ErrorResponse(fmt.Sprintf("Role %s is not configured", name)), nil
	}

	return nil, err
}

func (b *backend) pathRoleDelete(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	err := req.Storage.Delete(ctx, PathRole.For(data.Get("name").(string)))
	return nil, err
}

func (b *backend) pathRoleList(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	entries, err := req.Storage.List(ctx, PathRole.For(""))
	if err != nil {
		return nil, err
	}

	return logical.ListResponse(entries), nil
}
