package backend

import (
	"fmt"
	logicaltest "github.com/hashicorp/vault/helper/testhelpers/logical"
	"github.com/hashicorp/vault/sdk/logical"
	"testing"
)

func TestInfoRead(t *testing.T) {
	backend := testGetBackend(t)
	logicaltest.Test(t, logicaltest.TestCase{
		LogicalBackend: backend,
		Steps: []logicaltest.TestStep{
			{
				Operation: logical.ReadOperation,
				Path:      "info",
				ErrorOk:   false,
				Check: func(resp *logical.Response) error {
					for _, k := range []string{"version", "commit_sha", "description"} {
						if _, ok := resp.Data[k]; !ok {
							return fmt.Errorf("expected key %q is not present in response", k)
						}
					}

					return nil
				},
			},
		},
	})
}
