package backend

import (
	"context"
	"github.com/Shuttl-Tech/vault-plugin-postgres-cluster/version"
	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
)

func (b *backend) pathInfo(context.Context, *logical.Request, *framework.FieldData) (*logical.Response, error) {
	return &logical.Response{
		Data: map[string]interface{}{
			"description": "Manage credentials for dynamic fleets of PostgreSQL clusters",
			"commit_sha":  version.GitCommit,
			"version":     version.HumanVersion,
		},
	}, nil
}
