package backend

import (
	"context"
	"fmt"
	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
)

type Metadata struct {
	Cluster  string
	Database string
	Data     map[string]string
}

func (m *Metadata) Name() string {
	if m.Database != "" {
		return fmt.Sprintf("%s/%s", m.Cluster, m.Database)
	}

	return m.Cluster
}

func loadMetadataEntry(ctx context.Context, storage logical.Storage, addr string) (*Metadata, error) {
	entry, err := storage.Get(ctx, addr)
	if err != nil {
		return nil, err
	}

	if entry == nil {
		return nil, ErrNotFound
	}

	c := &Metadata{}
	err = entry.DecodeJSON(c)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func storeMetadataEntry(ctx context.Context, storage logical.Storage, addr string, cfg *Metadata) error {
	entry, err := logical.StorageEntryJSON(addr, cfg)
	if err != nil {
		return err
	}

	return storage.Put(ctx, entry)
}

func (b *backend) pathMetadataUpdate(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	cluster, ok := data.GetOk("cluster")
	if !ok {
		return logical.ErrorResponse("'cluster' attribute is required"), nil
	}

	_, err := loadClusterEntry(ctx, req.Storage, cluster.(string))
	if err == ErrNotFound {
		return logical.ErrorResponse(fmt.Sprintf("Cluster with name %s is not registered", cluster)), nil
	}

	if err != nil {
		return nil, err
	}

	addr := fmt.Sprintf("cluster/%s", cluster)

	database, ok := data.GetOk("database")
	if ok {
		_, err = loadDbEntry(ctx, req.Storage, cluster.(string), database.(string))
		if err == ErrNotFound {
			return logical.ErrorResponse(fmt.Sprintf("Database %q is not registered in cluster %q", database, cluster)), nil
		}

		if err != nil {
			return nil, err
		}

		addr = fmt.Sprintf("database/%s-%s", cluster, database)
	}

	meta, err := parseMetaAttr(data)
	if err != nil {
		return logical.ErrorResponse(err.Error()), nil
	}

	entry := &Metadata{
		Cluster: cluster.(string),
		Data:    meta,
	}

	if database != nil {
		entry.Database = database.(string)
	}

	err = storeMetadataEntry(ctx, req.Storage, PathMeta.For(addr), entry)
	if err != nil {
		return nil, err
	}

	return &logical.Response{
		Data: map[string]interface{}{
			"id": addr,
		},
	}, nil
}

func (b *backend) pathMetadataRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	for _, k := range []string{"database", "cluster"} {
		if _, ok := data.GetOk(k); ok {
			return logical.ErrorResponse(fmt.Sprintf("attribute %q is not supported on lookup", k)), nil
		}
	}

	tVal, ok := data.GetOk("type")
	if !ok {
		return logical.ErrorResponse("'type' attribute is required to perform lookup"), nil
	}

	target := tVal.(string)
	switch target {
	case "database", "cluster":
	default:
		return logical.ErrorResponse(fmt.Sprintf("invalid 'lookup' value %q, only 'cluster' or 'database' is supporter", target)), nil
	}

	meta, err := parseMetaAttr(data)
	if err != nil {
		return logical.ErrorResponse(err.Error()), nil
	}

	addr := fmt.Sprintf("%s/", target)

	objList, err := req.Storage.List(ctx, PathMeta.For(addr))
	if err != nil {
		return nil, err
	}

	var matches []string
	for _, key := range objList {
		m, err := loadMetadataEntry(ctx, req.Storage, PathMeta.For(addr+key))
		if err == ErrNotFound {
			continue
		}

		if err != nil {
			return nil, err
		}

		if matchAttrs(m.Data, meta) {
			matches = append(matches, m.Name())
		}
	}

	return logical.ListResponse(matches), nil
}

func matchAttrs(target, source map[string]string) bool {
	for sk, sv := range source {
		tv, ok := target[sk]
		if !ok {
			return false
		}

		if tv != sv {
			return false
		}
	}

	return true
}

func (b *backend) pathMetadataList(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	matches := map[string]interface{}{}
	clusters, err := req.Storage.List(ctx, PathMeta.For("cluster/"))
	if err != nil {
		return nil, err
	}

	for _, c := range clusters {
		cid := "cluster/" + c
		meta, err := loadMetadataEntry(ctx, req.Storage, PathMeta.For(cid))
		if err == ErrNotFound {
			continue
		}

		if err != nil {
			return nil, err
		}

		matches[cid] = meta.Data
	}

	databases, err := req.Storage.List(ctx, PathMeta.For("database/"))
	if err != nil {
		return nil, err
	}

	for _, d := range databases {
		did := "database/" + d
		meta, err := loadMetadataEntry(ctx, req.Storage, PathMeta.For(did))
		if err == ErrNotFound {
			continue
		}

		if err != nil {
			return nil, err
		}

		matches[did] = meta.Data
	}

	var keys []string
	for k := range matches {
		keys = append(keys, k)
	}

	return logical.ListResponseWithInfo(keys, matches), nil
}

func (b *backend) pathMetadataDelete(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	rid, ok := data.GetOk("id")
	if !ok {
		return logical.ErrorResponse("'id' is required to delete metadata"), nil
	}

	err := req.Storage.Delete(ctx, PathMeta.For(rid))
	if err != nil {
		return nil, err
	}

	return &logical.Response{}, nil
}

func parseMetaAttr(data *framework.FieldData) (map[string]string, error) {
	md, ok := data.GetOk("data")
	if !ok {
		return nil, fmt.Errorf("'data' attribute is required")
	}

	meta, ok := md.(map[string]string)
	if !ok {
		return nil, fmt.Errorf("value of 'data' must be key-value pairs")
	}

	if len(meta) < 1 {
		return nil, fmt.Errorf("'data' must contain at least one key-value pair")
	}

	return meta, nil
}
