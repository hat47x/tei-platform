//go:build tei_sqlite_reference && cgo

package effectfence

import (
	"context"
	"errors"
	"testing"

	"tei/internal/security/legacyauthorization"
)

func TestSQLiteExplicitEpochRevocationIsBusinessDenial(t *testing.T) {
	for _, dimension := range authorityDimensions {
		t.Run(dimension, func(t *testing.T) {
			a, b, identity, req := setupSQLiteLegacyEpochs(t)
			if err := b.SetAuthorityDependency(req.Scope, dimension, 2, false); err != nil {
				t.Fatal(err)
			}
			_, readErr := identity.ReadAuthorizationEpochs(context.Background(), legacyauthorization.Input{
				SubjectID: req.Scope.Subject, Action: req.Scope.Action, Resource: req.Scope.Resource,
			})
			if !errors.Is(readErr, legacyauthorization.ErrAuthorityInactive) {
				t.Fatalf("明示的な失効が専用エラーになりません: %v", readErr)
			}
			auth := fixtureLegacyAdapter(identity, nil)
			_, result, err := a.PrepareWithVersionedAuthorization(context.Background(), req.Scope.Subject, req, auth, fixtureAuthoritySource)
			if err != nil || result.Disposition != Denied || result.Reason != "authorization-denied" || result.Receipt != (Receipt{}) {
				t.Fatalf("失効時の準備が業務拒否になりません: %+v %v", result, err)
			}
			lookup, err := a.LookupWithVersionedAuthorization(context.Background(), req.Scope.Subject, req, auth, fixtureAuthoritySource)
			if err != nil || lookup.Status != ReceiptNotAllowed || lookup.Receipt != (Receipt{}) {
				t.Fatalf("失効時に記録を開示したか障害扱いしました: %+v %v", lookup, err)
			}
			if value, version := b.Observe(req.Scope); value != "" || version != 0 {
				t.Fatalf("失効後の副作用: %q %d", value, version)
			}
		})
	}
}

func TestSQLiteMissingEpochIsNotExplicitRevocation(t *testing.T) {
	for _, dimension := range authorityDimensions {
		t.Run(dimension, func(t *testing.T) {
			a, b, identity, req := setupSQLiteLegacyEpochs(t)
			if err := b.exec("DELETE FROM authority_dependencies WHERE scope=" + scopeSQL(req.Scope) +
				" AND dimension=" + textSQL(dimension)); err != nil {
				t.Fatal(err)
			}
			_, readErr := identity.ReadAuthorizationEpochs(context.Background(), legacyauthorization.Input{
				SubjectID: req.Scope.Subject, Action: req.Scope.Action, Resource: req.Scope.Resource,
			})
			if readErr == nil || errors.Is(readErr, legacyauthorization.ErrAuthorityInactive) {
				t.Fatalf("正本の欠落を失効として扱いました: %v", readErr)
			}
			auth := fixtureLegacyAdapter(identity, nil)
			_, result, err := a.PrepareWithVersionedAuthorization(context.Background(), req.Scope.Subject, req, auth, fixtureAuthoritySource)
			if err == nil || result != (Result{}) {
				t.Fatalf("正本欠落を業務拒否にしました: %+v %v", result, err)
			}
			lookup, err := a.LookupWithVersionedAuthorization(context.Background(), req.Scope.Subject, req, auth, fixtureAuthoritySource)
			if err == nil || lookup != (ReceiptLookupResult{}) {
				t.Fatalf("正本欠落を記録不存在にしました: %+v %v", lookup, err)
			}
		})
	}
}

func TestSQLiteMalformedEpochIsNotExplicitRevocation(t *testing.T) {
	a, b, identity, req := setupSQLiteLegacyEpochs(t)
	if err := b.exec("UPDATE authority_dependencies SET version=0,allowed=0 WHERE scope=" +
		scopeSQL(req.Scope) + " AND dimension=" + textSQL("membership")); err != nil {
		t.Fatal(err)
	}
	_, err := identity.ReadAuthorizationEpochs(context.Background(), legacyauthorization.Input{
		SubjectID: req.Scope.Subject, Action: req.Scope.Action, Resource: req.Scope.Resource,
	})
	if err == nil || errors.Is(err, legacyauthorization.ErrAuthorityInactive) {
		t.Fatalf("版破損を失効へ分類しました: %v", err)
	}
	result, err := a.LookupWithVersionedAuthorization(context.Background(), req.Scope.Subject, req,
		fixtureLegacyAdapter(identity, nil), fixtureAuthoritySource)
	if err == nil || result != (ReceiptLookupResult{}) {
		t.Fatalf("破損した版で記録の存在を判定しました: %+v %v", result, err)
	}
}

func TestSQLiteRevocationDuringRoleEvaluationIsDenial(t *testing.T) {
	a, b, identity, req := setupSQLiteLegacyEpochs(t)
	auth := fixtureLegacyAdapter(identity, func() {
		if err := b.SetAuthorityDependency(req.Scope, "policy", 2, false); err != nil {
			t.Fatal(err)
		}
	})
	_, result, err := a.PrepareWithVersionedAuthorization(context.Background(), req.Scope.Subject, req, auth, fixtureAuthoritySource)
	if err != nil || result.Disposition != Denied || result.Reason != "authorization-denied" {
		t.Fatalf("認可評価途中の失効を許可・実行障害にしました: %+v %v", result, err)
	}
	if _, version := a.Observe(req.Scope); version != 0 {
		t.Fatal("失効後に副作用が発生しました")
	}
}
