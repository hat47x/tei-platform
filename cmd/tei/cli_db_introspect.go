package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"tei/internal/core"
	"tei/internal/database/physical"
	"tei/internal/database/provider"
	"tei/internal/database/providers/postgres"
	"tei/internal/database/providers/sqlite"
	"tei/internal/database/schematosemantic"
	"tei/internal/environment"
)

func runDBIntrospect(ctx context.Context, providerName string, dsn string, out io.Writer, convertToSemantic bool) error {
	pid := environment.DatabaseProvider(strings.ToLower(strings.TrimSpace(providerName)))
	if !pid.Known() || pid == environment.ProviderGeneric {
		return fmt.Errorf("unsupported database provider %q", pid)
	}

	var snap physical.SchemaSnapshot
	var err error

	switch pid {
	case environment.ProviderPostgres:
		snap, err = introspectPostgres(ctx, dsn)
	case environment.ProviderSQLite:
		snap, err = introspectSQLite(ctx, dsn)
	default:
		return fmt.Errorf("introspection is not implemented for provider %q", pid)
	}

	if err != nil {
		return err
	}

	if convertToSemantic {
		result, err := schematosemantic.Project(snap, schematosemantic.DefaultOptions())
		if err != nil {
			return fmt.Errorf("semantic projection: %w", err)
		}
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(result); err != nil {
			return fmt.Errorf("encode semantic model: %w", err)
		}
		return nil
	}

	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	if err := enc.Encode(snap); err != nil {
		return fmt.Errorf("encode schema snapshot: %w", err)
	}
	return nil
}

func introspectPostgres(ctx context.Context, dsn string) (physical.SchemaSnapshot, error) {
	if strings.TrimSpace(dsn) == "" {
		return physical.SchemaSnapshot{}, errors.New("postgres DSN is required")
	}
	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		return physical.SchemaSnapshot{}, fmt.Errorf("connect: %w", err)
	}
	defer conn.Close()

	instance := environment.DatabaseInstance{
		ID:         core.MustCanonicalID(string(environment.ProviderPostgres)),
		Definition: core.MustCanonicalID(string(environment.ProviderPostgres)),
		Provider:   environment.ProviderPostgres,
	}
	p := postgres.Provider{Catalog: postgres.SQLCatalogReader{DB: conn}}
	return provider.Introspect(ctx, p, instance)
}

func introspectSQLite(ctx context.Context, dsn string) (physical.SchemaSnapshot, error) {
	if strings.TrimSpace(dsn) == "" {
		return physical.SchemaSnapshot{}, errors.New("sqlite DSN is required")
	}
	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return physical.SchemaSnapshot{}, fmt.Errorf("connect: %w", err)
	}
	defer conn.Close()

	instance := environment.DatabaseInstance{
		ID:         core.MustCanonicalID("sqlite.dev"),
		Definition: core.MustCanonicalID("sqlite"),
		Provider:   environment.ProviderSQLite,
	}
	p := sqlite.Provider{Catalog: sqlite.SQLCatalogReader{DB: conn}}
	return provider.Introspect(ctx, p, instance)
}
