package command

import (
	"fmt"
	"strings"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/ir"
	"tei/internal/semantic"
)

// dataEffectWrites reports whether an effect changes data. A capability with no
// writing effect is out of scope here.
func dataEffectWrites(kind semantic.DataEffectKind) bool {
	return kind == semantic.DataEffectCreate || kind == semantic.DataEffectUpdate || kind == semantic.DataEffectDelete
}

// checkDatasourceAccess refuses a writing command when the Application declares
// the data's database as read-only.
//
// The check is declaration-based: it reads the Application's datasource
// declarations and the database's semantic uses, and it does not observe the
// database itself. A datasource that covers the data but is declared
// read-only is enough to refuse the write, because the Application already said
// it does not write there. A data set that no datasource covers is not refused:
// absence of a declaration is not a permission, but guessing the database from
// a namespace would invent a binding the model does not have.
func checkDatasourceAccess(world ir.World, request Request, effect *semantic.DataEffect) error {
	if effect == nil || !dataEffectWrites(effect.Kind) {
		return nil
	}
	app, ok := applicationByID(world, request.Application)
	if !ok {
		return nil
	}
	namespace := dataNamespace(effect.Data)
	if namespace == "" {
		return nil
	}
	for _, requirement := range app.Databases {
		target, ok := databaseByID(world, requirement.TargetDatabase)
		if !ok {
			continue
		}
		if !databaseUsesNamespace(target, namespace) {
			continue
		}
		// 書込み可能な宣言で走査を止めない。要求は書込み先のdatasourceを
		// 指定しないため、同じdataを覆うread-only宣言を宣言順で隠してはならない(#2611)。
		if requirement.Access == database.AccessRead {
			return &ExecutionError{
				Code: "COMMAND_DATASOURCE_READ_ONLY",
				Message: fmt.Sprintf(
					"capability %s writes %s but application %s declares datasource %s (%s) as read-only",
					request.Capability, effect.Data, app.ID, requirement.Alias, requirement.TargetDatabase),
			}
		}
	}
	return nil
}

func applicationByID(world ir.World, id core.CanonicalID) (application.Application, bool) {
	for _, app := range world.Applications {
		if app.ID == id {
			return app, true
		}
	}
	return application.Application{}, false
}

func databaseByID(world ir.World, id core.CanonicalID) (database.Database, bool) {
	for _, db := range world.Databases {
		if db.ID == id {
			return db, true
		}
	}
	return database.Database{}, false
}

// databaseUsesNamespace reports whether the database declares the semantic
// namespace the data belongs to.
func databaseUsesNamespace(db database.Database, namespace core.CanonicalID) bool {
	for _, used := range db.SemanticUses {
		if used == namespace {
			return true
		}
	}
	return false
}

// dataNamespace returns the namespace part of a canonical data ID. A data ID
// without a namespace has none to compare, so the caller does not guess one.
func dataNamespace(data core.CanonicalID) core.CanonicalID {
	text := string(data)
	index := strings.LastIndex(text, ".")
	if index <= 0 {
		return ""
	}
	return core.CanonicalID(text[:index])
}
