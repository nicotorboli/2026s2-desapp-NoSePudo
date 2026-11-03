package dao_test

import (
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/dao"
)

func TestLedgerSqlNextSettlementGroupAvanza(t *testing.T) {
	db := startPostgres(t)
	ledgerDao := dao.NewLedgerDao(db)

	first, err := ledgerDao.NextSettlementGroup(t.Context(), nil)
	if err != nil {
		t.Fatalf("NextSettlementGroup devolvió %v", err)
	}
	second, err := ledgerDao.NextSettlementGroup(t.Context(), nil)
	if err != nil {
		t.Fatalf("NextSettlementGroup devolvió %v", err)
	}

	if second <= first {
		t.Errorf("la secuencia devolvió %d después de %d", second, first)
	}
}

func TestLedgerSqlInsertAsientosConSignoQueSumanCero(t *testing.T) {
	db := startPostgres(t)
	ledgerDao := dao.NewLedgerDao(db)
	userID := seedUser(t, db)

	settlementGroup, err := ledgerDao.NextSettlementGroup(t.Context(), nil)
	if err != nil {
		t.Fatalf("NextSettlementGroup devolvió %v", err)
	}

	entries := []model.LedgerEntry{
		{
			UserID:          &userID,
			Reason:          model.ReasonInitialGrant,
			Description:     "saldo inicial",
			SettlementGroup: settlementGroup,
			Amount:          model.CreditsFromUnits(1000),
		},
		{
			Reason:          model.ReasonInitialGrant,
			Description:     "saldo inicial, contrapartida de la plataforma",
			SettlementGroup: settlementGroup,
			Amount:          -model.CreditsFromUnits(1000),
		},
	}

	for _, entry := range entries {
		stored, insertErr := ledgerDao.Insert(t.Context(), nil, entry)
		if insertErr != nil {
			t.Fatalf("Insert devolvió %v", insertErr)
		}
		if stored.ID == 0 || stored.CreatedAt.IsZero() {
			t.Errorf("el asiento insertado es %+v", stored)
		}
	}

	unbalanced, err := ledgerDao.UnbalancedSettlementGroups(t.Context())
	if err != nil {
		t.Fatalf("UnbalancedSettlementGroups devolvió %v", err)
	}
	if len(unbalanced) != 0 {
		t.Errorf("hay grupos de liquidación que no suman cero: %v", unbalanced)
	}
}

func TestLedgerSqlUnbalancedSettlementGroupsDenunciaElDesbalance(t *testing.T) {
	db := startPostgres(t)
	ledgerDao := dao.NewLedgerDao(db)
	userID := seedUser(t, db)

	settlementGroup, err := ledgerDao.NextSettlementGroup(t.Context(), nil)
	if err != nil {
		t.Fatalf("NextSettlementGroup devolvió %v", err)
	}

	_, err = ledgerDao.Insert(t.Context(), nil, model.LedgerEntry{
		UserID:          &userID,
		Reason:          model.ReasonInitialGrant,
		Description:     "un asiento sin contrapartida",
		SettlementGroup: settlementGroup,
		Amount:          model.CreditsFromUnits(1000),
	})
	if err != nil {
		t.Fatalf("Insert devolvió %v", err)
	}

	unbalanced, err := ledgerDao.UnbalancedSettlementGroups(t.Context())
	if err != nil {
		t.Fatalf("UnbalancedSettlementGroups devolvió %v", err)
	}
	if len(unbalanced) != 1 || unbalanced[0] != settlementGroup {
		t.Errorf("se denunciaron los grupos %v, se esperaba sólo el %d", unbalanced, settlementGroup)
	}
}

func TestLedgerSqlInsertRechazaUnAsientoDeCero(t *testing.T) {
	db := startPostgres(t)
	ledgerDao := dao.NewLedgerDao(db)
	userID := seedUser(t, db)

	settlementGroup, err := ledgerDao.NextSettlementGroup(t.Context(), nil)
	if err != nil {
		t.Fatalf("NextSettlementGroup devolvió %v", err)
	}

	_, err = ledgerDao.Insert(t.Context(), nil, model.LedgerEntry{
		UserID:          &userID,
		Reason:          model.ReasonInitialGrant,
		Description:     "un asiento de cero",
		SettlementGroup: settlementGroup,
		Amount:          0,
	})
	if err == nil {
		t.Error("la base aceptó un asiento de importe cero")
	}
}
