package teiparse

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// World is a simplified representation of the TEI canonical world.
type World struct {
	Namespaces   []Namespace
	DataModels   []DataModel
	Capabilities []Capability
	Applications []Application
	Queries      []Query
	Databases    []*Database
	Migrations   []*Migration
}

// Namespace represents a TEI namespace declaration.
type Namespace struct {
	Name string
}

// DataModel represents a TEI data definition.
type DataModel struct {
	Namespace string
	Name      string
	Types     []TypeDef
	Attributes []Attribute
	Relations []Relation
}

// TypeDef represents a type alias.
type TypeDef struct {
	Name string
	From string
}

// Attribute represents a data attribute.
type Attribute struct {
	Name string
	Type string
}

// Relation represents a relationship.
type Relation struct {
	Name      string
	Target    string
	Cardinality string
	Keys      []string
}

// Capability represents a capability definition.
type Capability struct {
	ID      string
	Input   string
	Output  string
}

// Application represents a TEI application.
type Application struct {
	ID           string
	SemanticUses []string
	Validations  []string
	Databases    []DatabaseRequirement
	Patterns     []Pattern
}

// DatabaseRequirement represents a database requirement.
type DatabaseRequirement struct {
	Target string
	Access string
}

// Pattern represents an application pattern.
type Pattern struct {
	ID        string
	Kind      string
	Data      string
	Query     string
	Capability string
}

// Query represents an embedded SQL query.
type Query struct {
	ID    string
	SQL   string
	AppID string
}

// Database represents a TEI database definition.
type Database struct {
	Name       string
	Kind       string
	Ownership  string
	UsesSemantic []string
}

// Migration represents a database migration.
type Migration struct {
	ID        string
	Database  string
	Recovery  string
	SchemaSQL string
	DataSQL   string
	VerifySQL string
}

// ParseWorld parses a TEI workspace directory into a World.
func ParseWorld(workspaceRoot string) (*World, error) {
	world := &World{}

	semanticDir := filepath.Join(workspaceRoot, "semantic")
	if err := parseSemanticDir(semanticDir, world); err != nil {
		return nil, fmt.Errorf("parse semantic: %w", err)
	}

	appDir := filepath.Join(workspaceRoot, "applications")
	if err := parseApplicationDir(appDir, world); err != nil {
		return nil, fmt.Errorf("parse applications: %w", err)
	}

	queryDir :=filepath.Join(workspaceRoot, "queries")
	if err := parseQueryDir(queryDir, world); err != nil {
		return nil, fmt.Errorf("parse queries: %w", err)
	}

	dbDir := filepath.Join(workspaceRoot, "databases")
	if err := parseDatabaseDir(dbDir, world); err != nil {
		return nil, fmt.Errorf("parse databases: %w", err)
	}

	return world, nil
}

func parseDatabaseDir(dir string, world *World) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			if err := parseDatabaseDir(path, world); err != nil {
				return err
			}
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".tei") {
			continue
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		if strings.HasSuffix(entry.Name(), "migration.tei") {
			if err := parseMigrationFile(f, world); err != nil {
				f.Close()
				return fmt.Errorf("%s: %w", path, err)
			}
		} else {
			if err := parseDatabaseFile(f, world); err != nil {
				f.Close()
				return fmt.Errorf("%s: %w", path, err)
			}
		}
		f.Close()
	}
	return nil
}

func parseDatabaseFile(f *os.File, world *World) error {
	scanner := bufio.NewScanner(f)
	var db *Database
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		if strings.HasPrefix(line, "database ") {
			name := strings.TrimSpace(strings.TrimPrefix(line, "database "))
			name = strings.TrimRight(name, " {")
			db = &Database{Name: name}
			continue
		}
		if db == nil {
			continue
		}
		if strings.HasPrefix(line, "kind ") {
			db.Kind = strings.TrimSpace(strings.TrimPrefix(line, "kind "))
		} else if strings.HasPrefix(line, "ownership ") {
			db.Ownership = strings.TrimSpace(strings.TrimPrefix(line, "ownership "))
		} else if strings.HasPrefix(line, "uses semantic ") {
			db.UsesSemantic = append(db.UsesSemantic, strings.TrimSpace(strings.TrimPrefix(line, "uses semantic ")))
		}
	}
	if db != nil {
		world.Databases = append(world.Databases, db)
	}
	return scanner.Err()
}

func parseMigrationFile(f *os.File, world *World) error {
	scanner := bufio.NewScanner(f)
	var mig *Migration
	var inSchema, inData, inVerify bool
	var currentSQL strings.Builder
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		if strings.HasPrefix(line, "migration ") {
			id := strings.TrimSpace(strings.TrimPrefix(line, "migration "))
			id = strings.TrimRight(id, " {")
			mig = &Migration{ID: id}
			world.Migrations = append(world.Migrations, mig)
			continue
		}
		if mig == nil {
			continue
		}
		if strings.HasPrefix(line, "database ") {
			mig.Database = strings.TrimSpace(strings.TrimPrefix(line, "database "))
			continue
		}
		if strings.HasPrefix(line, "recovery ") {
			mig.Recovery = strings.TrimSpace(strings.TrimPrefix(line, "recovery "))
			continue
		}
		if strings.HasPrefix(line, "schema sql ") {
			inSchema = true
			inData = false
			inVerify = false
			currentSQL.Reset()
			continue
		}
		if strings.HasPrefix(line, "data sql ") {
			inSchema = false
			inData = true
			inVerify = false
			currentSQL.Reset()
			continue
		}
		if strings.HasPrefix(line, "verify sql ") {
			inSchema = false
			inData = false
			inVerify = true
			currentSQL.Reset()
			continue
		}
		if strings.HasPrefix(line, "}") {
			sql := strings.TrimSpace(currentSQL.String())
			if inSchema {
				mig.SchemaSQL = sql
			} else if inData {
				mig.DataSQL = sql
			} else if inVerify {
				mig.VerifySQL = sql
			}
			inSchema = false
			inData = false
			inVerify = false
			currentSQL.Reset()
			continue
		}
		if inSchema || inData || inVerify {
			if currentSQL.Len() > 0 {
				currentSQL.WriteString("\n")
			}
			currentSQL.WriteString(line)
			continue
		}
	}
	return scanner.Err()
}

func parseSemanticDir(dir string, world *World) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			if err := parseSemanticDir(filepath.Join(dir, entry.Name()), world); err != nil {
				return err
			}
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".tei") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		if err := parseSemanticFile(f, world); err != nil {
			f.Close()
			return fmt.Errorf("%s: %w", path, err)
		}
		f.Close()
	}
	return nil
}

func parseSemanticFile(f *os.File, world *World) error {
	scanner := bufio.NewScanner(f)
	var currentNamespace string
	var currentModel *DataModel
	var currentCap *Capability

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		if strings.HasPrefix(line, "namespace ") {
			currentNamespace = strings.TrimSpace(strings.TrimPrefix(line, "namespace "))
			world.Namespaces = append(world.Namespaces, Namespace{Name: currentNamespace})
			continue
		}

		if strings.HasPrefix(line, "type ") {
			rest := strings.TrimPrefix(line, "type ")
			parts := strings.Fields(rest)
			if len(parts) >= 3 && parts[1] == "from" {
				if currentModel != nil {
					currentModel.Types = append(currentModel.Types, TypeDef{Name: parts[0], From: strings.Join(parts[2:], " ")})
				}
			}
			continue
		}

		if strings.HasPrefix(line, "data ") {
			name := strings.TrimSpace(strings.TrimPrefix(line, "data "))
			currentModel = &DataModel{Namespace: currentNamespace, Name: name}
			world.DataModels = append(world.DataModels, *currentModel)
			continue
		}

		if strings.HasPrefix(line, "capability ") {
			rest := strings.TrimPrefix(line, "capability ")
			id := strings.TrimSpace(rest)
			currentCap = &Capability{ID: id}
			world.Capabilities = append(world.Capabilities, *currentCap)
			continue
		}

		if strings.HasPrefix(line, "}") {
			if currentCap != nil {
				currentCap = nil
			}
			if currentModel != nil {
				currentModel = nil
			}
			continue
		}

		if currentModel != nil {
			if strings.HasPrefix(line, "attribute ") {
				attrLine := strings.TrimPrefix(line, "attribute ")
				parts := strings.Fields(attrLine)
				if len(parts) >= 3 && parts[1] == "type" {
					world.DataModels[len(world.DataModels)-1].Attributes = append(
						world.DataModels[len(world.DataModels)-1].Attributes,
						Attribute{Name: parts[0], Type: strings.Join(parts[2:], " ")},
					)
				}
			} else if strings.HasPrefix(line, "relation ") {
				relLine := strings.TrimPrefix(line, "relation ")
				parts := strings.Fields(relLine)
				if len(parts) >= 3 {
					world.DataModels[len(world.DataModels)-1].Relations = append(
						world.DataModels[len(world.DataModels)-1].Relations,
						Relation{Name: parts[0], Target: parts[1], Cardinality: parts[2]},
					)
				}
			}
		}

		if currentCap != nil {
			if strings.HasPrefix(line, "input ") {
				currentCap.Input = strings.TrimSpace(strings.TrimPrefix(line, "input "))
			} else if strings.HasPrefix(line, "output ") {
				currentCap.Output = strings.TrimSpace(strings.TrimPrefix(line, "output "))
			}
		}
	}
	return scanner.Err()
}

func parseApplicationDir(dir string, world *World) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			if err := parseApplicationDir(filepath.Join(dir, entry.Name()), world); err != nil {
				return err
			}
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".tei") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		if err := parseApplicationFile(f, world); err != nil {
			f.Close()
			return fmt.Errorf("%s: %w", path, err)
		}
		f.Close()
	}
	return nil
}

func parseApplicationFile(f *os.File, world *World) error {
	scanner := bufio.NewScanner(f)
	var currentApp *Application

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		if strings.HasPrefix(line, "application ") {
			id := strings.TrimSpace(strings.TrimPrefix(line, "application "))
			currentApp = &Application{ID: id}
			world.Applications = append(world.Applications, *currentApp)
			continue
		}

		if strings.HasPrefix(line, "}") {
			currentApp = nil
			continue
		}

		if currentApp != nil {
			if strings.HasPrefix(line, "uses semantic ") {
				world.Applications[len(world.Applications)-1].SemanticUses = append(
					world.Applications[len(world.Applications)-1].SemanticUses,
					strings.TrimSpace(strings.TrimPrefix(line, "uses semantic ")),
				)
			} else if strings.HasPrefix(line, "uses validation ") {
				world.Applications[len(world.Applications)-1].Validations = append(
					world.Applications[len(world.Applications)-1].Validations,
					strings.TrimSpace(strings.TrimPrefix(line, "uses validation ")),
				)
			} else if strings.HasPrefix(line, "database ") {
				parts := strings.Fields(line)
				if len(parts) >= 4 {
					world.Applications[len(world.Applications)-1].Databases = append(
						world.Applications[len(world.Applications)-1].Databases,
						DatabaseRequirement{Target: parts[1], Access: parts[3]},
					)
				}
			} else if strings.HasPrefix(line, "pattern ") {
				patLine := strings.TrimPrefix(line, "pattern ")
				parts := strings.Fields(patLine)
				if len(parts) >= 3 {
					world.Applications[len(world.Applications)-1].Patterns = append(
						world.Applications[len(world.Applications)-1].Patterns,
						Pattern{ID: parts[0], Kind: parts[1], Data: parts[2]},
					)
				}
			} else if strings.HasPrefix(line, "command ") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					world.Applications[len(world.Applications)-1].Patterns = append(
						world.Applications[len(world.Applications)-1].Patterns,
						Pattern{ID: parts[0], Kind: "command", Capability: parts[1]},
					)
				}
			}
		}
	}
	return scanner.Err()
}

func parseQueryDir(dir string, world *World) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			if err := parseQueryDir(filepath.Join(dir, entry.Name()), world); err != nil {
				return err
			}
			continue
		}
		path := filepath.Join(dir, entry.Name())
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".sql", ".tei":
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			sql, err := readNonEmptyLines(f)
			f.Close()
			if err != nil {
				return err
			}
			id := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
			world.Queries = append(world.Queries, Query{ID: id, SQL: strings.Join(sql, "\n")})
		}
	}
	return nil
}

func readNonEmptyLines(f *os.File) ([]string, error) {
	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "--") {
			lines = append(lines, line)
		}
	}
	return lines, scanner.Err()
}

// FindApplication finds an application by ID.
func (w *World) FindApplication(id string) *Application {
	for i := range w.Applications {
		if strings.EqualFold(w.Applications[i].ID, id) {
			return &w.Applications[i]
		}
	}
	return nil
}

// FindDataModel finds a data model by name across namespaces.
func (w *World) FindDataModel(name string) *DataModel {
	for i := range w.DataModels {
		if strings.EqualFold(w.DataModels[i].Name, name) {
			return &w.DataModels[i]
		}
	}
	return nil
}
