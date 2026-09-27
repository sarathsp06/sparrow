package accessauth

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/sarathsp06/sparrow/pkg/access"
	"github.com/sarathsp06/sparrow/pkg/access/memstore"
	"github.com/sarathsp06/sparrow/pkg/access/pgstore"
	"github.com/sarathsp06/sparrow/pkg/access/storetest"
)

// The migration must ship exactly the library's schema, or pgstore queries
// could drift from the tables Sparrow creates.
func TestMigrationMatchesPgstoreSchema(t *testing.T) {
	b, err := os.ReadFile("../../db/migrations/000027_access_tokens.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), pgstore.Schema) {
		t.Fatal("db/migrations/000027_access_tokens.up.sql no longer contains pkg/access/pgstore/schema.sql verbatim; copy it over")
	}
}

func TestTTLRules(t *testing.T) {
	acme := "acme"
	cases := []struct {
		consumer  *string
		requested time.Duration
		want      time.Duration
		wantErr   bool
	}{
		{nil, 0, 0, false}, // tenant-wide: never expires by default
		{nil, 90 * 24 * time.Hour, 90 * 24 * time.Hour, false},
		{nil, 10 * 365 * 24 * time.Hour, 10 * 365 * 24 * time.Hour, false},
		{&acme, 0, ConsumerTokenDefaultTTL, false},
		{&acme, time.Hour, time.Hour, false},
		{&acme, ConsumerTokenMaxTTL + time.Second, 0, true},
	}
	for _, c := range cases {
		got, err := TokenTTL(c.consumer, c.requested)
		if got != c.want || (err != nil) != c.wantErr {
			t.Errorf("TokenTTL(%v, %v) = %v, %v", c.consumer, c.requested, got, err)
		}
	}
	if got, _ := InviteTTL(0); got != InviteDefaultTTL {
		t.Errorf("InviteTTL(0) = %v", got)
	}
	if _, err := InviteTTL(InviteMaxTTL + time.Second); err == nil || err.Error() != "invites can live at most 7 days" {
		t.Errorf("InviteTTL above max: %v", err)
	}
	if _, err := TokenTTL(&acme, ConsumerTokenMaxTTL+time.Second); err == nil || err.Error() != "consumer tokens can live at most 30 days" {
		t.Errorf("TokenTTL above max: %v", err)
	}
}

func TestMasterKeyIsRootInDefaultTenant(t *testing.T) {
	svc, err := NewWithStore(memstore.New(), "master")
	if err != nil {
		t.Fatal(err)
	}
	p, err := svc.Authenticate(context.Background(), "master")
	if err != nil || !p.Root || p.Realm != Realm() || p.Name != MasterKeyName {
		t.Fatalf("master key principal = %+v, %v", p, err)
	}

	open, _ := NewWithStore(memstore.New(), "")
	if _, err := open.Authenticate(context.Background(), ""); err == nil {
		t.Fatal("empty credential authenticated without a master key")
	}
	_, secret, err := svc.CreateToken(context.Background(), access.CreateTokenRequest{Realm: Realm(), Name: "x", CreatedBy: MasterKeyName})
	if err != nil || !strings.HasPrefix(secret, TokenPrefix) {
		t.Fatalf("token secret %q, %v", secret, err)
	}
}

// TestPgstoreConformance runs the library's store contract against real
// Postgres, in a throwaway schema. Needs DATABASE_URL (CI provides one).
func TestPgstoreConformance(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set")
	}
	admin, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close() //nolint:errcheck
	if err := admin.Ping(); err != nil {
		t.Skipf("postgres unreachable: %v", err)
	}

	n := 0
	storetest.Run(t, func(t *testing.T) access.Store {
		n++
		schema := fmt.Sprintf("access_conformance_%d_%d", time.Now().UnixNano(), n)
		if _, err := admin.Exec(`CREATE SCHEMA ` + schema); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _, _ = admin.Exec(`DROP SCHEMA ` + schema + ` CASCADE`) })

		sep := "?"
		if strings.Contains(url, "?") {
			sep = "&"
		}
		db, err := sql.Open("pgx", url+sep+"search_path="+schema)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = db.Close() })
		if _, err := db.Exec(pgstore.Schema); err != nil {
			t.Fatal(err)
		}
		return pgstore.New(db)
	})
}

func TestValidateConsumer(t *testing.T) {
	for _, ok := range []string{"acme", "team-a", "Team_B.2", "ünïcode", strings.Repeat("x", 255)} {
		if err := ValidateConsumer(ok); err != nil {
			t.Errorf("ValidateConsumer(%q) = %v", ok, err)
		}
	}
	for _, bad := range []string{"", " acme", "acme ", ".", "..", "../../v1", "a/b", `a\b`, "a?b", "a#b", "a%2fb", "a\nb", strings.Repeat("x", 256)} {
		if err := ValidateConsumer(bad); err == nil {
			t.Errorf("ValidateConsumer(%q) accepted", bad)
		}
	}
}
