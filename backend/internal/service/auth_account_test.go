package service_test

import (
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

func (f *marketFixture) accountOf(t *testing.T, email string) (model.Account, bool) {
	t.Helper()

	var (
		account model.Account
		id      int64
	)

	err := f.db.QueryRowContext(t.Context(),
		`SELECT a.user_id, a.available_balance, a.reserved_balance
		 FROM user_accounts a JOIN users u ON u.id = a.user_id
		 WHERE u.email = $1`, email,
	).Scan(&id, &account.AvailableBalance, &account.ReservedBalance)
	if err != nil {
		return model.Account{}, false
	}
	account.UserID = id

	return account, true
}

func (f *marketFixture) initialGrantEntries(t *testing.T, userID int64) int {
	t.Helper()

	var total int
	if err := f.db.QueryRowContext(t.Context(),
		`SELECT COUNT(*) FROM ledger_entries
		 WHERE reason = 'INITIAL_GRANT'
		   AND settlement_group IN (
		       SELECT settlement_group FROM ledger_entries WHERE user_id = $1 AND reason = 'INITIAL_GRANT'
		   )`, userID,
	).Scan(&total); err != nil {
		t.Fatalf("no se pudieron contar los asientos del crédito inicial: %v", err)
	}

	return total
}

func TestAltaDeUsuarioAbreSuCuentaConElSaldoInicialYSuAsiento(t *testing.T) {
	fixture := newMarketFixture(t)
	saldo := model.CreditsFromUnits(1000)
	registration := fixture.registrationRepository(saldo)

	const email = "nuevo@test.local"

	user, err := registration.InsertUserWithAccount(t.Context(), model.User{
		Email:        email,
		PasswordHash: marketFakeHash,
		Privilege:    model.PrivilegeUser,
		Active:       true,
	})
	if err != nil {
		t.Fatalf("InsertUserWithAccount devolvió %v", err)
	}
	if user.ID == 0 {
		t.Fatal("el usuario quedó sin identificador")
	}

	account, found := fixture.accountOf(t, email)
	if !found {
		t.Fatal("el alta no abrió una cuenta de mercado")
	}
	if account.AvailableBalance != saldo {
		t.Errorf("la cuenta quedó con %d, se esperaban %d créditos", account.AvailableBalance, saldo)
	}
	if account.ReservedBalance != 0 {
		t.Errorf("la cuenta nació con %d créditos reservados, se esperaba cero", account.ReservedBalance)
	}

	if entries := fixture.initialGrantEntries(t, user.ID); entries != 2 {
		t.Errorf("el crédito inicial dejó %d asientos, se esperaban 2: el usuario y la contrapartida", entries)
	}
	if !fixture.ledgerIsBalanced(t) {
		t.Error("el crédito inicial no suma cero")
	}
}

func TestAltaConSaldoCeroAbreLaCuentaSinAsientos(t *testing.T) {
	fixture := newMarketFixture(t)
	registration := fixture.registrationRepository(0)

	const email = "sin-credito@test.local"

	user, err := registration.InsertUserWithAccount(t.Context(), model.User{
		Email:        email,
		PasswordHash: marketFakeHash,
		Privilege:    model.PrivilegeUser,
		Active:       true,
	})
	if err != nil {
		t.Fatalf("InsertUserWithAccount devolvió %v", err)
	}

	account, found := fixture.accountOf(t, email)
	if !found {
		t.Fatal("el alta sin crédito no abrió una cuenta")
	}
	if account.AvailableBalance != 0 {
		t.Errorf("la cuenta quedó con %d, se esperaba cero", account.AvailableBalance)
	}
	if entries := fixture.initialGrantEntries(t, user.ID); entries != 0 {
		t.Errorf("se escribieron %d asientos por un crédito de cero: un asiento en cero viola la restricción", entries)
	}
}

func TestElAltaDuplicadaNoDejaCuentaHuerfana(t *testing.T) {
	fixture := newMarketFixture(t)
	registration := fixture.registrationRepository(model.CreditsFromUnits(1000))

	const email = "repetido@test.local"

	nuevo := model.User{
		Email:        email,
		PasswordHash: marketFakeHash,
		Privilege:    model.PrivilegeUser,
		Active:       true,
	}

	if _, err := registration.InsertUserWithAccount(t.Context(), nuevo); err != nil {
		t.Fatalf("el primer alta devolvió %v", err)
	}

	if _, err := registration.InsertUserWithAccount(t.Context(), nuevo); err == nil {
		t.Fatal("el segundo alta con el mismo email fue aceptado")
	}

	var cuentas int
	if err := fixture.db.QueryRowContext(t.Context(),
		`SELECT COUNT(*) FROM user_accounts a JOIN users u ON u.id = a.user_id WHERE u.email = $1`, email,
	).Scan(&cuentas); err != nil {
		t.Fatalf("no se pudieron contar las cuentas: %v", err)
	}
	if cuentas != 1 {
		t.Errorf("hay %d cuentas para el email, se esperaba 1: el alta rechazada se deshace entera", cuentas)
	}

	if !fixture.ledgerIsBalanced(t) {
		t.Error("el libro mayor quedó desbalanceado tras un alta rechazada")
	}
}
