package backend

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/hashicorp/vault/logical"
	logicaltest "github.com/hashicorp/vault/logical/testing"
	"github.com/mitchellh/mapstructure"
	"path"
	"strings"
	"testing"
	"time"
)

const (
	testCluster         = "test-role-cluster"
	testDb              = "test-role-db"
	testRole            = "test-role-role"
	testCredsDefaultTTL = 60
	testCredsMaxTTL     = 120
)

type T struct {
	*testing.T
	isRevokeTest bool
}

func (t *T) Error(args ...interface{}) {
	if t.isRevokeTest {
		arg := fmt.Sprint(args...)
		if strings.Contains(arg, "Revoke error:") ||
			strings.Contains(arg, "WARNING: Revoking the following secret failed. It may") {
			t.T.Logf("[DEBUG] suppressing the error returned by revoke operation. %s", arg)
			return
		}
	} else {
		t.T.Error(args...)
	}
}

func TestAccCredsCreate(t *testing.T) {
	config := logical.TestBackendConfig()
	config.StorageView = &logical.InmemStorage{}
	backend, err := Factory(context.Background(), config)
	if err != nil {
		t.Errorf("Failed to initialize backend factory. %s", err)
		return
	}

	cleanup, clusterAttr := prepareTestContainer(t)
	defer cleanup()

	rolesAttr := map[string]interface{}{
		"default_ttl": testCredsDefaultTTL,
		"max_ttl":     testCredsMaxTTL,
	}

	cluster := &ClusterConfig{}
	testStorage := &logical.InmemStorage{}

	// Vault testing framework manages the storage provided to vault core
	// we want to make sure that if we make any request out of band we get
	// the data in storage
	err = storeClusterEntry(context.Background(), testStorage, testCluster, cluster)
	if err != nil {
		t.Errorf("failed to store cluster entry in test storage: %s", err)
	}

	hijackT := &T{T: t}
	logicaltest.Test(hijackT, logicaltest.TestCase{
		LogicalBackend: backend,
		Steps: []logicaltest.TestStep{
			testAccWriteClusterConfig(t, path.Join("cluster", testCluster), clusterAttr, false),
			testAccReadClusterConfigVar(t, path.Join("cluster", testCluster), cluster),
			testAccWriteDbConfig(t, path.Join("cluster", testCluster, testDb)),
			testAccReadDbConfigCopy(t, path.Join("cluster", testCluster, testDb), testStorage),
			testAccWriteRoleConfig(t, path.Join("roles", testRole), rolesAttr, false),
			testAccReadRoleConfigCopy(t, path.Join("roles", testRole), testStorage),
			testAccReadCreds(hijackT, cluster, backend, testStorage),
		},
	})
}

func testAccReadCreds(t *T, cluster *ClusterConfig, backend logical.Backend, storage logical.Storage) logicaltest.TestStep {
	return logicaltest.TestStep{
		Operation: logical.ReadOperation,
		Path:      path.Join("creds", testCluster, testDb, testRole),
		ErrorOk:   false,
		Check: logicaltest.TestCheckMulti(
			testAccCheckCreds(cluster),
			testAccCheckRenewCreds(t, cluster, backend, storage),
			testAccCheckRevokeCreds(t, cluster, backend, storage),
		),
	}
}

func testAccCheckCreds(cluster *ClusterConfig) logicaltest.TestCheckFunc {
	return func(resp *logical.Response) error {
		if resp == nil {
			return fmt.Errorf("expected non nil response")
		}

		if resp.Secret == nil {
			return fmt.Errorf("no secrets available in response")
		}

		u, ok := resp.Data["username"]
		if !ok {
			return fmt.Errorf("username is not available in secret response")
		}

		p, ok := resp.Data["password"]
		if !ok {
			return fmt.Errorf("password is not available in secret response")
		}

		conn, err := sql.Open("postgres", fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", u, p, cluster.Host, cluster.Port, testDb))
		if err != nil {
			return fmt.Errorf("failed to connect with database using provisioned creds: %s", err)
		}
		defer conn.Close()

		_, err = conn.Exec(`create table testing (name varchar(64))`)
		if err != nil {
			return fmt.Errorf("failed to create table using provisioned creds: %s", err)
		}

		return nil
	}
}

func testAccCheckRenewCreds(t *T, cluster *ClusterConfig, backend logical.Backend, storage logical.Storage) logicaltest.TestCheckFunc {
	// This function is second in line so we don't
	// need to perform any sanity check that has been
	// done by testAccCheckCreds
	return func(resp *logical.Response) error {
		u, p := resp.Data["username"], resp.Data["password"]
		conn, err := sql.Open("postgres", fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", u, p, cluster.Host, cluster.Port, testDb))
		if err != nil {
			return fmt.Errorf("failed to connect with database using provisioned creds: %s", err)
		}
		defer conn.Close()

		var (
			vResp  time.Time
			vDb    time.Time
			vnResp time.Time
			vnDb   time.Time
			tmp    []byte
		)

		vResp = resp.Secret.ExpirationTime().UTC()

		err = conn.QueryRow(`select valuntil from pg_user where usename = $1`, u).Scan(&tmp)
		if err != nil {
			return fmt.Errorf("failed to retrieve password expiry. %s", err)
		}

		vDb, err = time.Parse("2006-01-02 15:04:05+00", string(tmp))
		if err != nil {
			return fmt.Errorf("failed to unmarshal time data. %s", err)
		}

		if vResp.Truncate(time.Second) != vDb.UTC().Truncate(time.Second) {
			return fmt.Errorf("valid_until on response does not match with db. response(%s) != db(%s)", vResp.UTC(), vDb.UTC())
		}

		// Lease has a maximum precision of one second,
		// this will make sure we don't end up renewing the
		// creds on same second as the initial creation time.
		time.Sleep(2 * time.Second)

		renResp, err := backend.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.RenewOperation,
			Storage:   storage,
			Secret: &logical.Secret{
				InternalData: map[string]interface{}{
					"secret_type": SecretCredsType,
					"username":    u,
					"role":        testRole,
					"cluster":     testCluster,
					"database":    testDb,
				},
			},
		})

		if err != nil {
			return fmt.Errorf("failed to renew credentials. internal error: %s", err)
		}

		if renResp.IsError() {
			return fmt.Errorf("failed to renew credentials. response error: %s", renResp.Error())
		}

		vnResp = resp.Secret.ExpirationTime().UTC()
		err = conn.QueryRow(`select valuntil from pg_user where usename = $1`, u).Scan(&tmp)
		if err != nil {
			return fmt.Errorf("failed to retrieve new password expiry. %s", err)
		}

		vnDb, err = time.Parse("2006-01-02 15:04:05+00", string(tmp))
		if err != nil {
			return fmt.Errorf("failed to unmarshal new time data. %s", err)
		}

		if vnResp.Add(5*time.Second).Truncate(time.Minute) != vnDb.UTC().Truncate(time.Minute) {
			return fmt.Errorf("new valid_until on response does not match with db. response(%s) != db(%s)", vnResp.UTC(), vnDb.UTC())
		}

		return nil
	}
}

func testAccCheckRevokeCreds(t *T, cluster *ClusterConfig, backend logical.Backend, storage logical.Storage) logicaltest.TestCheckFunc {
	return func(resp *logical.Response) error {
		t.isRevokeTest = true

		u, p := resp.Data["username"], resp.Data["password"]
		conn, err := sql.Open("postgres", fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", u, p, cluster.Host, cluster.Port, testDb))
		if err != nil {
			return fmt.Errorf("failed to connect with database using provisioned creds: %s", err)
		}
		defer conn.Close()

		_, err = conn.Exec(`create table users (id serial primary key)`)
		if err != nil {
			return fmt.Errorf("failed to create new table. %s", err)
		}

		renResp, err := backend.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.RevokeOperation,
			Storage:   storage,
			Secret: &logical.Secret{
				InternalData: map[string]interface{}{
					"secret_type": SecretCredsType,
					"username":    u,
					"role":        testRole,
					"cluster":     testCluster,
					"database":    testDb,
				},
			},
		})

		if err != nil {
			return fmt.Errorf("failed to revoke lease. internal error: %s", err)
		}

		if renResp.IsError() {
			return fmt.Errorf("failed to revoke lease. response error: %s", renResp.Error())
		}

		_, err = conn.Exec(`create table profiles (id serial primary key)`)
		if err == nil {
			return fmt.Errorf("expected error on query execution using revoked credentials. cerds are still valid")
		}

		if !strings.HasSuffix(err.Error(), " was concurrently dropped") {
			return fmt.Errorf("unexpected error message: %s", err)
		}

		invalidConn, _ := sql.Open("postgres", fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", u, p, cluster.Host, cluster.Port, testDb))
		err = invalidConn.Ping()
		if err == nil {
			_ = invalidConn.Close()
			return fmt.Errorf("expected error when trying to connect using revoked credentials. connection is valid")
		}

		if !strings.HasSuffix(err.Error(), fmt.Sprintf("password authentication failed for user %q", u)) {
			return fmt.Errorf("unexpected error message: %s", err)
		}

		return nil
	}
}

func testAccReadDbConfigCopy(t *testing.T, name string, storage logical.Storage) logicaltest.TestStep {
	return logicaltest.TestStep{
		Operation: logical.ReadOperation,
		Path:      name,
		ErrorOk:   false,
		Check: func(resp *logical.Response) error {
			dbc := &DbConfig{}
			err := mapstructure.Decode(resp.Data, dbc)
			if err != nil {
				return fmt.Errorf("failed to decode database configuration. %s", err)
			}

			return storeDbEntry(context.Background(), storage, testCluster, testDb, dbc)
		},
	}
}

func testAccReadRoleConfigCopy(t *testing.T, sourceName string, storage logical.Storage) logicaltest.TestStep {
	return logicaltest.TestStep{
		Operation: logical.ReadOperation,
		Path:      sourceName,
		ErrorOk:   false,
		Check: func(resp *logical.Response) error {
			role := &RoleConfig{}
			err := mapstructure.Decode(resp.Data, role)
			if err != nil {
				return fmt.Errorf("failed to decode role into object. %s", err)
			}

			return storeRoleEntry(context.Background(), storage, testRole, role)
		},
	}
}
