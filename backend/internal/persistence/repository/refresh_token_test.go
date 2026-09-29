package repository_test

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/dao"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/repository"
)

// refreshTokenFixture trae los dos repositorios y una cuenta ya creada: las
// credenciales tienen una clave foránea contra users, así que no hay forma de
// probarlas sin una cuenta real.
type refreshTokenFixture struct {
	repo *repository.RefreshTokenRepository
	db *sql.DB
	userID int64
}

func newRefreshTokenFixture(t *testing.T) refreshTokenFixture {
	t.Helper()

	db := startPostgres(t)

	user, err := repository.NewUserRepository(dao.NewUserDao(db)).
		Insert(t.Context(), aUser("nico@nosepudo.ar"))
	if err != nil {
		t.Fatalf("no se pudo crear la cuenta: %v", err)
	}

	return refreshTokenFixture{
		repo: repository.NewRefreshTokenRepository(dao.NewRefreshTokenDao(db)),
		db: db,
		userID: user.ID,
	}
}

func (f refreshTokenFixture) aToken(familyID string) model.RefreshToken {
	return model.RefreshToken{
		ExpiresAt: time.Now().Add(168 * time.Hour),
		ID: uuid.NewString(),
		FamilyID: familyID,
		UserID: f.userID,
	}
}

// insertLive deja una credencial viva y la devuelve.
func (f refreshTokenFixture) insertLive(t *testing.T, familyID string) model.RefreshToken {
	t.Helper()

	token := f.aToken(familyID)
	if err := f.repo.Insert(t.Context(), token); err != nil {
		t.Fatalf("no se pudo insertar la credencial: %v", err)
	}

	return token
}

func (f refreshTokenFixture) countLive(t *testing.T) int {
	t.Helper()

	var count int
	if err := f.db.QueryRowContext(t.Context(),
		"SELECT COUNT(*) FROM refresh_tokens WHERE user_id = $1 AND used_at IS NULL AND revoked_at IS NULL",
		f.userID,
	).Scan(&count); err != nil {
		t.Fatalf("no se pudieron contar las credenciales vivas: %v", err)
	}

	return count
}

func TestRefreshTokenInsertAndReadBack(t *testing.T) {
	fixture := newRefreshTokenFixture(t)

	inserted := fixture.insertLive(t, uuid.NewString())

	found, err := fixture.repo.GetByID(t.Context(), inserted.ID)
	if err != nil {
		t.Fatalf("GetByID devolvió error: %v", err)
	}

	if found.ID != inserted.ID {
		t.Errorf("ID = %q, se esperaba %q", found.ID, inserted.ID)
	}
	if found.FamilyID != inserted.FamilyID {
		t.Errorf("FamilyID = %q", found.FamilyID)
	}
	if found.UserID != fixture.userID {
		t.Errorf("UserID = %d, se esperaba %d", found.UserID, fixture.userID)
	}
	if found.IssuedAt.IsZero() {
		t.Error("la base no asignó issued_at")
	}
	if !found.IsLive(time.Now()) {
		t.Error("la credencial recién insertada no está viva")
	}
	if found.UsedAt != nil || found.RevokedAt != nil {
		t.Error("la credencial nace usada o revocada")
	}
}

func TestRefreshTokenGetByIDReportsAMiss(t *testing.T) {
	fixture := newRefreshTokenFixture(t)

	_, err := fixture.repo.GetByID(t.Context(), uuid.NewString())
	if !errors.Is(err, model.ErrRefreshTokenNotFound) {
		t.Errorf("GetByID devolvió %v, se esperaba ErrRefreshTokenNotFound", err)
	}
}

// el requerimiento: la rotación deja la presentada usada y su reemplazo vivo, heredando
// la misma familia.
func TestRefreshTokenRotateConsumesTheOldAndIssuesTheNew(t *testing.T) {
	fixture := newRefreshTokenFixture(t)
	family := uuid.NewString()

	original := fixture.insertLive(t, family)
	replacement := fixture.aToken(family)

	if err := fixture.repo.Rotate(t.Context(), original.ID, replacement); err != nil {
		t.Fatalf("Rotate devolvió error: %v", err)
	}

	consumed, err := fixture.repo.GetByID(t.Context(), original.ID)
	if err != nil {
		t.Fatalf("la credencial original desapareció: %v", err)
	}
	if consumed.UsedAt == nil {
		t.Error("la credencial original no quedó marcada como usada")
	}
	if consumed.IsLive(time.Now()) {
		t.Error("la credencial original sigue viva")
	}

	fresh, err := fixture.repo.GetByID(t.Context(), replacement.ID)
	if err != nil {
		t.Fatalf("el reemplazo no se insertó: %v", err)
	}
	if !fresh.IsLive(time.Now()) {
		t.Error("el reemplazo no está vivo")
	}
	if fresh.FamilyID != family {
		t.Errorf("el reemplazo tiene familia %q, se esperaba que heredara %q", fresh.FamilyID, family)
	}

	if live := fixture.countLive(t); live != 1 {
		t.Errorf("hay %d credenciales vivas, se esperaba 1", live)
	}
}

// el requerimiento: presentar una credencial ya usada se rechaza. El UPDATE lleva las
// condiciones en su WHERE, así que es esa operación la que decide y no un
// chequeo previo.
func TestRefreshTokenRotateRefusesAConsumedToken(t *testing.T) {
	fixture := newRefreshTokenFixture(t)
	family := uuid.NewString()

	original := fixture.insertLive(t, family)
	if err := fixture.repo.Rotate(t.Context(), original.ID, fixture.aToken(family)); err != nil {
		t.Fatalf("la primera rotación devolvió error: %v", err)
	}

	// Segundo uso de la misma credencial.
	replayReplacement := fixture.aToken(family)
	err := fixture.repo.Rotate(t.Context(), original.ID, replayReplacement)

	if !errors.Is(err, model.ErrRefreshTokenReused) {
		t.Fatalf("Rotate devolvió %v, se esperaba ErrRefreshTokenReused", err)
	}

	// Y la transacción no dejó nada: el reemplazo del replay no existe.
	if _, err := fixture.repo.GetByID(t.Context(), replayReplacement.ID); !errors.Is(err, model.ErrRefreshTokenNotFound) {
		t.Error("la rotación rechazada insertó su reemplazo igual")
	}
}

// Una rotación que falla no escribe nada, que es lo que la transacción
// garantiza: el reemplazo no puede quedar sin que la original se consuma.
func TestRefreshTokenRotateWritesNothingWhenItFails(t *testing.T) {
	fixture := newRefreshTokenFixture(t)
	family := uuid.NewString()

	revoked := fixture.insertLive(t, family)
	if err := fixture.repo.RevokeFamily(t.Context(), family, fixture.userID); err != nil {
		t.Fatalf("RevokeFamily devolvió error: %v", err)
	}

	replacement := fixture.aToken(family)
	if err := fixture.repo.Rotate(t.Context(), revoked.ID, replacement); !errors.Is(err, model.ErrRefreshTokenReused) {
		t.Fatalf("Rotate devolvió %v, se esperaba ErrRefreshTokenReused", err)
	}

	if _, err := fixture.repo.GetByID(t.Context(), replacement.ID); !errors.Is(err, model.ErrRefreshTokenNotFound) {
		t.Error("se insertó el reemplazo de una rotación que falló")
	}
	if live := fixture.countLive(t); live != 0 {
		t.Errorf("hay %d credenciales vivas, se esperaba 0", live)
	}
}

// Una credencial expirada tampoco rota: el WHERE incluye expires_at.
func TestRefreshTokenRotateRefusesAnExpiredToken(t *testing.T) {
	fixture := newRefreshTokenFixture(t)
	family := uuid.NewString()

	expired := fixture.aToken(family)
	expired.ExpiresAt = time.Now().Add(-time.Hour)
	if err := fixture.repo.Insert(t.Context(), expired); err != nil {
		t.Fatalf("no se pudo insertar la credencial expirada: %v", err)
	}

	err := fixture.repo.Rotate(t.Context(), expired.ID, fixture.aToken(family))
	if !errors.Is(err, model.ErrRefreshTokenReused) {
		t.Errorf("Rotate devolvió %v, se esperaba que rechazara la expirada", err)
	}
}

// el requerimiento y el caso borde: cerrar una sesión no cierra las otras.
func TestRefreshTokenRevokeFamilyTouchesOneFamilyOnly(t *testing.T) {
	fixture := newRefreshTokenFixture(t)

	phone := uuid.NewString()
	desktop := uuid.NewString()

	phoneToken := fixture.insertLive(t, phone)
	desktopToken := fixture.insertLive(t, desktop)

	if err := fixture.repo.RevokeFamily(t.Context(), phone, fixture.userID); err != nil {
		t.Fatalf("RevokeFamily devolvió error: %v", err)
	}

	revoked, err := fixture.repo.GetByID(t.Context(), phoneToken.ID)
	if err != nil {
		t.Fatalf("GetByID devolvió error: %v", err)
	}
	if revoked.RevokedAt == nil {
		t.Error("la credencial de la familia cerrada no quedó revocada")
	}

	survivor, err := fixture.repo.GetByID(t.Context(), desktopToken.ID)
	if err != nil {
		t.Fatalf("GetByID devolvió error: %v", err)
	}
	if !survivor.IsLive(time.Now()) {
		t.Error("cerrar una sesión cerró la otra")
	}

	if live := fixture.countLive(t); live != 1 {
		t.Errorf("hay %d credenciales vivas, se esperaba 1", live)
	}
}

// el criterio: la respuesta al robo deja cero credenciales usables en la cuenta, y
// alcanza a todas las familias y no sólo a la afectada.
func TestRefreshTokenRevokeAllLiveForUserLeavesNoneUsable(t *testing.T) {
	fixture := newRefreshTokenFixture(t)

	for range 3 {
		fixture.insertLive(t, uuid.NewString())
	}
	if live := fixture.countLive(t); live != 3 {
		t.Fatalf("el caso arrancó con %d credenciales vivas, se esperaban 3", live)
	}

	if err := fixture.repo.RevokeAllLiveForUser(t.Context(), fixture.userID); err != nil {
		t.Fatalf("RevokeAllLiveForUser devolvió error: %v", err)
	}

	if live := fixture.countLive(t); live != 0 {
		t.Errorf("quedaron %d credenciales vivas, se esperaban 0", live)
	}
}

// La revocación no toca las credenciales de otras cuentas.
func TestRefreshTokenRevokeAllLiveForUserLeavesOtherAccountsAlone(t *testing.T) {
	fixture := newRefreshTokenFixture(t)

	other, err := repository.NewUserRepository(dao.NewUserDao(fixture.db)).
		Insert(t.Context(), aUser("otro@nosepudo.ar"))
	if err != nil {
		t.Fatalf("no se pudo crear la segunda cuenta: %v", err)
	}

	fixture.insertLive(t, uuid.NewString())

	otherToken := model.RefreshToken{
		ExpiresAt: time.Now().Add(168 * time.Hour),
		ID: uuid.NewString(),
		FamilyID: uuid.NewString(),
		UserID: other.ID,
	}
	if err = fixture.repo.Insert(t.Context(), otherToken); err != nil {
		t.Fatalf("no se pudo insertar la credencial de la otra cuenta: %v", err)
	}

	if err = fixture.repo.RevokeAllLiveForUser(t.Context(), fixture.userID); err != nil {
		t.Fatalf("RevokeAllLiveForUser devolvió error: %v", err)
	}

	survivor, err := fixture.repo.GetByID(t.Context(), otherToken.ID)
	if err != nil {
		t.Fatalf("GetByID devolvió error: %v", err)
	}
	if !survivor.IsLive(time.Now()) {
		t.Error("se revocó la credencial de otra cuenta")
	}
}

// Sólo una de dos rotaciones concurrentes de la misma credencial puede ganar.
// Es la carrera que el WHERE del UPDATE cierra, y la razón por la que la
// decisión no puede tomarse con un SELECT previo.
func TestRefreshTokenConcurrentRotationsLeaveOneWinner(t *testing.T) {
	fixture := newRefreshTokenFixture(t)
	family := uuid.NewString()

	original := fixture.insertLive(t, family)

	const attempts = 4
	results := make(chan error, attempts)

	for range attempts {
		go func() {
			results <- fixture.repo.Rotate(t.Context(), original.ID, fixture.aToken(family))
		}()
	}

	succeeded, reused := 0, 0
	for range attempts {
		switch err := <-results; {
		case err == nil:
			succeeded++
		case errors.Is(err, model.ErrRefreshTokenReused):
			reused++
		default:
			t.Errorf("una rotación falló por otro motivo: %v", err)
		}
	}

	if succeeded != 1 {
		t.Errorf("%d rotaciones tuvieron éxito, se esperaba exactamente 1", succeeded)
	}
	if reused != attempts-1 {
		t.Errorf("%d rotaciones se rechazaron por reuso, se esperaban %d", reused, attempts-1)
	}
	if live := fixture.countLive(t); live != 1 {
		t.Errorf("hay %d credenciales vivas, se esperaba 1", live)
	}
}

