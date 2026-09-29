package service_test

import (
	"errors"
	"testing"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

// signIn deja una cuenta creada y una sesión abierta, y devuelve la sesión junto
// con el jti de su credencial de renovación, que es lo que la renovación
// presenta.
func (f authFixture) signIn(t *testing.T, email, password string) (model.Session, string) {
	t.Helper()

	if _, err := f.auth.Register(t.Context(), email, password); err != nil {
		t.Fatalf("no se pudo crear la cuenta: %v", err)
	}

	session, err := f.auth.Login(t.Context(), email, password)
	if err != nil {
		t.Fatalf("no se pudo iniciar sesión: %v", err)
	}

	return session, f.liveTokenID(t)
}

// liveTokenID devuelve el jti de la única credencial viva. Si hubiera más de
// una, el caso está mal armado y conviene enterarse.
func (f authFixture) liveTokenID(t *testing.T) string {
	t.Helper()

	var found string
	for id, token := range f.refreshTokens.byID {
		if token.IsLive(f.clock.now()) {
			if found != "" {
				t.Fatal("hay más de una credencial viva y el caso esperaba una")
			}
			found = id
		}
	}
	if found == "" {
		t.Fatal("no hay ninguna credencial de renovación viva")
	}

	return found
}

func (f authFixture) sessionFamily(t *testing.T, tokenID string) string {
	t.Helper()

	token, found := f.refreshTokens.byID[tokenID]
	if !found {
		t.Fatalf("no existe la credencial %q", tokenID)
	}

	return token.FamilyID
}

// el requerimiento: el inicio de sesión entrega las dos credenciales y dice cuándo vence
// cada una, y la de renovación queda persistida para poder cortarla.
func TestLoginIssuesBothCredentialsAndPersistsTheRefreshOne(t *testing.T) {
	fixture := newAuthFixture()

	session, _ := fixture.signIn(t, "nico@nosepudo.ar", "una-contraseña")

	if session.AccessToken == "" {
		t.Error("no se emitió credencial de acceso")
	}
	if session.RefreshToken == "" {
		t.Error("no se emitió credencial de renovación")
	}
	if !session.RefreshExpiresAt.After(session.AccessExpiresAt) {
		t.Errorf("la renovación (%v) no vive más que el acceso (%v)",
			session.RefreshExpiresAt, session.AccessExpiresAt)
	}
	if live := fixture.refreshTokens.countLive(); live != 1 {
		t.Errorf("hay %d credenciales vivas persistidas, se esperaba 1", live)
	}
}

// el requerimiento y el requerimiento: la renovación entrega una sesión nueva y consume la anterior.
func TestRefreshRotatesTheCredential(t *testing.T) {
	fixture := newAuthFixture()
	session, tokenID := fixture.signIn(t, "nico@nosepudo.ar", "una-contraseña")
	family := fixture.sessionFamily(t, tokenID)

	renewed, err := fixture.auth.Refresh(t.Context(), tokenID, 1, family)
	if err != nil {
		t.Fatalf("Refresh devolvió error: %v", err)
	}

	if renewed.AccessToken == "" {
		t.Error("la renovación no emitió credencial de acceso")
	}
	if renewed.RefreshToken == session.RefreshToken {
		t.Error("la renovación devolvió la misma credencial de renovación")
	}

	// La presentada quedó consumida y sigue existiendo: la fila usada es la
	// evidencia que permite detectar un replay.
	if consumed := fixture.refreshTokens.byID[tokenID]; !consumed.IsConsumed() {
		t.Error("la credencial presentada no quedó consumida")
	}

	if live := fixture.refreshTokens.countLive(); live != 1 {
		t.Errorf("hay %d credenciales vivas, se esperaba 1", live)
	}
	if fresh := fixture.liveTokenID(t); fixture.sessionFamily(t, fresh) != family {
		t.Error("el reemplazo no heredó la familia de la sesión")
	}
}

// el requerimiento y el criterio: un segundo uso se rechaza y deja cero credenciales usables
// en la cuenta, no sólo en la familia afectada.
func TestRefreshReuseRevokesEveryCredentialOfTheAccount(t *testing.T) {
	fixture := newAuthFixture()
	_, firstTokenID := fixture.signIn(t, "nico@nosepudo.ar", "una-contraseña")
	family := fixture.sessionFamily(t, firstTokenID)

	// Una segunda sesión, como si fuera otro dispositivo.
	if _, err := fixture.auth.Login(t.Context(), "nico@nosepudo.ar", "una-contraseña"); err != nil {
		t.Fatalf("el segundo inicio de sesión devolvió error: %v", err)
	}

	// La primera credencial se usa una vez, legítimamente.
	if _, err := fixture.auth.Refresh(t.Context(), firstTokenID, 1, family); err != nil {
		t.Fatalf("la primera renovación devolvió error: %v", err)
	}

	// Y una segunda vez, que es lo que se lee como robo.
	_, err := fixture.auth.Refresh(t.Context(), firstTokenID, 1, family)

	if !errors.Is(err, model.ErrRefreshTokenReused) {
		t.Fatalf("Refresh devolvió %v, se esperaba ErrRefreshTokenReused", err)
	}
	if live := fixture.refreshTokens.countLive(); live != 0 {
		t.Errorf("quedaron %d credenciales vivas, se esperaban 0", live)
	}
	if len(fixture.refreshTokens.revokedAllFor) != 1 {
		t.Errorf("la revocación total corrió %d veces, se esperaba 1",
			len(fixture.refreshTokens.revokedAllFor))
	}
}

// Una credencial cuya fila no existe se trata como reuso: bien firmada pero sin
// fila es una que ya se consumió, o una emitida con una clave que se perdió.
func TestRefreshTreatsAMissingRowAsReuse(t *testing.T) {
	fixture := newAuthFixture()
	_, tokenID := fixture.signIn(t, "nico@nosepudo.ar", "una-contraseña")
	family := fixture.sessionFamily(t, tokenID)

	_, err := fixture.auth.Refresh(t.Context(), "un-jti-que-no-existe", 1, family)

	if !errors.Is(err, model.ErrRefreshTokenReused) {
		t.Errorf("Refresh devolvió %v, se esperaba ErrRefreshTokenReused", err)
	}
	if live := fixture.refreshTokens.countLive(); live != 0 {
		t.Errorf("quedaron %d credenciales vivas tras detectar el reuso", live)
	}
}

// El quinto escenario de la historia: expirada no es robada. El titular vuelve
// a entrar y el resto de sus sesiones no se toca.
func TestRefreshExpiredDoesNotTriggerTheTheftResponse(t *testing.T) {
	fixture := newAuthFixture()
	_, tokenID := fixture.signIn(t, "nico@nosepudo.ar", "una-contraseña")
	family := fixture.sessionFamily(t, tokenID)

	// Una semana y un minuto después.
	fixture.clock.instant = serviceNow.Add(168*time.Hour + time.Minute)

	_, err := fixture.auth.Refresh(t.Context(), tokenID, 1, family)

	if !errors.Is(err, model.ErrRefreshTokenExpired) {
		t.Fatalf("Refresh devolvió %v, se esperaba ErrRefreshTokenExpired", err)
	}
	if errors.Is(err, model.ErrRefreshTokenReused) {
		t.Error("una credencial expirada se reportó como reuso")
	}
	if len(fixture.refreshTokens.revokedAllFor) != 0 {
		t.Error("una credencial expirada disparó la respuesta al robo")
	}
}

// El privilegio se relee de la cuenta al renovar: una sesión larga no puede
// congelar un nivel viejo.
func TestRefreshRereadsThePrivilegeFromTheAccount(t *testing.T) {
	fixture := newAuthFixture()
	_, tokenID := fixture.signIn(t, "nico@nosepudo.ar", "una-contraseña")
	family := fixture.sessionFamily(t, tokenID)

	promoted := fixture.userRepository.byEmail["nico@nosepudo.ar"]
	promoted.Privilege = model.PrivilegeSuperuser
	fixture.userRepository.byEmail["nico@nosepudo.ar"] = promoted

	if _, err := fixture.auth.Refresh(t.Context(), tokenID, promoted.ID, family); err != nil {
		t.Fatalf("Refresh devolvió error: %v", err)
	}

	if fixture.tokenIssuer.gotPrivilege != model.PrivilegeSuperuser {
		t.Errorf("la credencial nueva se emitió con privilegio %v, se esperaba el actual de la cuenta",
			fixture.tokenIssuer.gotPrivilege)
	}
}

// el requerimiento y el caso borde: cerrar sesión corta esa sesión y no las
// de los otros dispositivos.
func TestLogoutRevokesOnlyItsOwnFamily(t *testing.T) {
	fixture := newAuthFixture()
	_, phoneTokenID := fixture.signIn(t, "nico@nosepudo.ar", "una-contraseña")
	phoneFamily := fixture.sessionFamily(t, phoneTokenID)

	if _, err := fixture.auth.Login(t.Context(), "nico@nosepudo.ar", "una-contraseña"); err != nil {
		t.Fatalf("el segundo inicio de sesión devolvió error: %v", err)
	}

	if err := fixture.auth.Logout(t.Context(), 1, phoneFamily); err != nil {
		t.Fatalf("Logout devolvió error: %v", err)
	}

	if revoked := fixture.refreshTokens.byID[phoneTokenID]; revoked.RevokedAt == nil {
		t.Error("la credencial de la sesión cerrada no quedó revocada")
	}
	if live := fixture.refreshTokens.countLive(); live != 1 {
		t.Errorf("quedaron %d credenciales vivas, se esperaba 1: la de la otra sesión", live)
	}
}

// el criterio: después de cerrar sesión, renovar con esa credencial se rechaza.
func TestRefreshAfterLogoutIsRefused(t *testing.T) {
	fixture := newAuthFixture()
	_, tokenID := fixture.signIn(t, "nico@nosepudo.ar", "una-contraseña")
	family := fixture.sessionFamily(t, tokenID)

	if err := fixture.auth.Logout(t.Context(), 1, family); err != nil {
		t.Fatalf("Logout devolvió error: %v", err)
	}

	_, err := fixture.auth.Refresh(t.Context(), tokenID, 1, family)
	if !errors.Is(err, model.ErrRefreshTokenRevoked) {
		t.Errorf("Refresh devolvió %v, se esperaba ErrRefreshTokenRevoked", err)
	}
	if errors.Is(err, model.ErrRefreshTokenReused) {
		t.Error("cerrar sesión y reintentar se trató como robo: eso cortaría las otras sesiones")
	}
	if len(fixture.refreshTokens.revokedAllFor) != 0 {
		t.Error("reintentar con una credencial revocada disparó la respuesta al robo")
	}
}

// Si la revocación falla al responder a un robo, se reporta ese error y no se
// lo tapa con el de reuso: quedarse con credenciales vivas después de detectar
// un robo es peor que responder 500.
func TestRefreshReportsAFailedRevocation(t *testing.T) {
	fixture := newAuthFixture()
	_, tokenID := fixture.signIn(t, "nico@nosepudo.ar", "una-contraseña")
	family := fixture.sessionFamily(t, tokenID)

	if _, err := fixture.auth.Refresh(t.Context(), tokenID, 1, family); err != nil {
		t.Fatalf("la primera renovación devolvió error: %v", err)
	}

	revocationFailure := errors.New("la base no responde")
	fixture.refreshTokens.revokeAllErr = revocationFailure

	_, err := fixture.auth.Refresh(t.Context(), tokenID, 1, family)

	if !errors.Is(err, revocationFailure) {
		t.Errorf("Refresh devolvió %v, se esperaba que propagara el fallo de la revocación", err)
	}
}

// Si la rotación pierde la carrera contra otra petición, el resultado es el
// mismo que un reuso: la decisión la toma el UPDATE y no el chequeo previo.
func TestRefreshLosingTheRotationRaceIsTreatedAsReuse(t *testing.T) {
	fixture := newAuthFixture()
	_, tokenID := fixture.signIn(t, "nico@nosepudo.ar", "una-contraseña")
	family := fixture.sessionFamily(t, tokenID)

	fixture.refreshTokens.rotateErr = model.ErrRefreshTokenReused

	_, err := fixture.auth.Refresh(t.Context(), tokenID, 1, family)

	if !errors.Is(err, model.ErrRefreshTokenReused) {
		t.Errorf("Refresh devolvió %v, se esperaba ErrRefreshTokenReused", err)
	}
	if len(fixture.refreshTokens.revokedAllFor) != 1 {
		t.Error("perder la carrera no disparó la respuesta al robo")
	}
}
