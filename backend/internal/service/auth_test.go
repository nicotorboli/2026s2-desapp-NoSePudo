package service_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/adapters"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/service"
)

// mockUserRepository guarda las cuentas en memoria, indexadas por el email ya
// normalizado, que es exactamente lo que hace la columna con su índice único.
type mockUserRepository struct {
	byEmail      map[string]model.User
	insertErr    error
	getByEmailer func(email string) (model.User, error)
	inserted     []model.User
	nextID       int64
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{byEmail: map[string]model.User{}, nextID: 1}
}

func (m *mockUserRepository) Insert(_ context.Context, user model.User) (model.User, error) {
	if m.insertErr != nil {
		return model.User{}, m.insertErr
	}
	if _, taken := m.byEmail[user.Email]; taken {
		return model.User{}, model.ErrEmailTaken
	}

	user.ID = m.nextID
	m.nextID++
	m.byEmail[user.Email] = user
	m.inserted = append(m.inserted, user)

	return user, nil
}

func (m *mockUserRepository) GetByEmail(_ context.Context, email string) (model.User, error) {
	if m.getByEmailer != nil {
		return m.getByEmailer(email)
	}
	if user, found := m.byEmail[email]; found {
		return user, nil
	}
	return model.User{}, model.ErrUserNotFound
}

func (m *mockUserRepository) GetByID(_ context.Context, id int64) (model.User, error) {
	for _, user := range m.byEmail {
		if user.ID == id {
			return user, nil
		}
	}
	return model.User{}, model.ErrUserNotFound
}

// mockPasswordHasher no hashea: antepone un prefijo. Alcanza para comprobar
// que el service guarda lo que el adaptador devolvió y no el texto plano, y
// evita pagar un bcrypt real por caso.
type mockPasswordHasher struct {
	hashErr       error
	hashed        []string
	dummyCompares int
}

const hashPrefix = "hashed:"

func (m *mockPasswordHasher) Hash(plain string) (string, error) {
	if m.hashErr != nil {
		return "", m.hashErr
	}
	m.hashed = append(m.hashed, plain)
	return hashPrefix + plain, nil
}

func (m *mockPasswordHasher) Compare(hash, plain string) error {
	if hash == hashPrefix+plain {
		return nil
	}
	return errors.New("no coinciden")
}

func (m *mockPasswordHasher) CompareWithDummy(string) {
	m.dummyCompares++
}

// mockTokenIssuer devuelve una credencial reconocible y registra con qué se la
// pidieron, que es lo que el service tiene que haber sacado de la cuenta.
type mockTokenIssuer struct {
	expiresAt    time.Time
	err          error
	sessionIDs   []string
	gotSubject   int64
	issued       int
	gotPrivilege model.PrivilegeLevel
}

func (m *mockTokenIssuer) IssueAccess(
	subject int64,
	privilege model.PrivilegeLevel,
	sessionID string,
) (string, adapters.Claims, error) {
	m.issued++
	m.gotSubject = subject
	m.gotPrivilege = privilege
	m.sessionIDs = append(m.sessionIDs, sessionID)

	if m.err != nil {
		return "", adapters.Claims{}, m.err
	}

	return fmt.Sprintf("access-token-de-%d", subject), adapters.Claims{
		ExpiresAt: m.expiresAt,
		SessionID: sessionID,
		Subject:   subject,
		Privilege: privilege,
		Kind:      adapters.KindAccess,
	}, nil
}

func newAuthService() (*service.Auth, *mockUserRepository, *mockPasswordHasher) {
	auth, userRepository, passwordHasher, _ := newAuthServiceWithIssuer()
	return auth, userRepository, passwordHasher
}

func newAuthServiceWithIssuer() (*service.Auth, *mockUserRepository, *mockPasswordHasher, *mockTokenIssuer) {
	userRepository := newMockUserRepository()
	passwordHasher := &mockPasswordHasher{}
	tokenIssuer := &mockTokenIssuer{expiresAt: time.Date(2026, time.September, 28, 12, 15, 0, 0, time.UTC)}

	return service.NewAuthService(userRepository, passwordHasher, tokenIssuer),
		userRepository, passwordHasher, tokenIssuer
}

func TestRegisterStoresANormalizedEmailAndAHash(t *testing.T) {
	auth, userRepository, passwordHasher := newAuthService()

	user, err := auth.Register(t.Context(), "  Nico@NoSePudo.AR ", "una-contraseña")
	if err != nil {
		t.Fatalf("Register devolvió error: %v", err)
	}

	if user.Email != "nico@nosepudo.ar" {
		t.Errorf("Email = %q, se esperaba normalizado", user.Email)
	}
	if user.ID == 0 {
		t.Error("la cuenta creada no tiene identificador")
	}
	if user.PasswordHash != hashPrefix+"una-contraseña" {
		t.Errorf("PasswordHash = %q, se esperaba lo que devolvió el adaptador", user.PasswordHash)
	}
	if strings.Contains(user.PasswordHash, "una-contraseña") && user.PasswordHash == "una-contraseña" {
		t.Error("se guardó la contraseña en texto plano")
	}
	if len(passwordHasher.hashed) != 1 {
		t.Errorf("se hasheó %d veces, se esperaba 1", len(passwordHasher.hashed))
	}
	if stored := userRepository.byEmail["nico@nosepudo.ar"]; stored.ID != user.ID {
		t.Error("la cuenta no quedó guardada bajo su email normalizado")
	}
}

// FR-022: una cuenta autoregistrada es siempre usuario común. No hay parámetro
// ni campo por el que un cliente pueda pedir otra cosa.
func TestRegisterAlwaysCreatesACommonUser(t *testing.T) {
	auth, _, _ := newAuthService()

	user, err := auth.Register(t.Context(), "nico@nosepudo.ar", "una-contraseña")
	if err != nil {
		t.Fatalf("Register devolvió error: %v", err)
	}

	if user.Privilege != model.PrivilegeUser {
		t.Errorf("Privilege = %v, se esperaba PrivilegeUser", user.Privilege)
	}
	if !user.Active {
		t.Error("la cuenta se creó inactiva")
	}
}

func TestRegisterRefusesATakenEmail(t *testing.T) {
	auth, userRepository, _ := newAuthService()

	first, err := auth.Register(t.Context(), "nico@nosepudo.ar", "una-contraseña")
	if err != nil {
		t.Fatalf("el primer alta devolvió error: %v", err)
	}

	_, err = auth.Register(t.Context(), "nico@nosepudo.ar", "otra-contraseña")
	if !errors.Is(err, model.ErrEmailTaken) {
		t.Fatalf("el segundo alta devolvió %v, se esperaba ErrEmailTaken", err)
	}

	if len(userRepository.inserted) != 1 {
		t.Errorf("se insertaron %d cuentas, se esperaba 1", len(userRepository.inserted))
	}
	if stored := userRepository.byEmail["nico@nosepudo.ar"]; stored.ID != first.ID {
		t.Error("la cuenta existente fue modificada")
	}
}

// El caso borde de la spec: dos formas de la misma dirección son una sola
// cuenta, porque las dos se normalizan igual antes de consultar.
func TestRegisterTreatsCaseAndWhitespaceAsTheSameAddress(t *testing.T) {
	auth, userRepository, _ := newAuthService()

	if _, err := auth.Register(t.Context(), "nico@nosepudo.ar", "una-contraseña"); err != nil {
		t.Fatalf("el primer alta devolvió error: %v", err)
	}

	_, err := auth.Register(t.Context(), "  NICO@NoSePudo.AR  ", "otra-contraseña")
	if !errors.Is(err, model.ErrEmailTaken) {
		t.Errorf("devolvió %v, se esperaba ErrEmailTaken", err)
	}
	if len(userRepository.inserted) != 1 {
		t.Errorf("una dirección se volvió %d cuentas", len(userRepository.inserted))
	}
}

// Si la consulta previa falla por algo que no es "no existe", eso no puede
// leerse como "el email está libre" y seguir de largo.
func TestRegisterPropagatesALookupFailure(t *testing.T) {
	auth, userRepository, passwordHasher := newAuthService()
	lookupFailure := errors.New("la base no responde")
	userRepository.getByEmailer = func(string) (model.User, error) {
		return model.User{}, lookupFailure
	}

	_, err := auth.Register(t.Context(), "nico@nosepudo.ar", "una-contraseña")

	if !errors.Is(err, lookupFailure) {
		t.Errorf("Register devolvió %v, se esperaba que propagara el fallo", err)
	}
	if len(userRepository.inserted) != 0 {
		t.Error("se creó una cuenta pese a que la consulta previa falló")
	}
	if len(passwordHasher.hashed) != 0 {
		t.Error("se hasheó la contraseña pese a que la consulta previa falló")
	}
}

// El índice único es el que decide de verdad: aunque la consulta previa diga
// que está libre, el INSERT puede chocar, y ese error tiene que llegar como
// ErrEmailTaken y no como un 500.
func TestRegisterSurfacesTheUniqueIndexViolation(t *testing.T) {
	auth, userRepository, _ := newAuthService()
	userRepository.getByEmailer = func(string) (model.User, error) {
		return model.User{}, model.ErrUserNotFound
	}
	userRepository.insertErr = model.ErrEmailTaken

	_, err := auth.Register(t.Context(), "nico@nosepudo.ar", "una-contraseña")

	if !errors.Is(err, model.ErrEmailTaken) {
		t.Errorf("Register devolvió %v, se esperaba ErrEmailTaken", err)
	}
}

func TestRegisterDoesNotCreateAnAccountWhenHashingFails(t *testing.T) {
	auth, userRepository, passwordHasher := newAuthService()
	passwordHasher.hashErr = errors.New("bcrypt no pudo")

	_, err := auth.Register(t.Context(), "nico@nosepudo.ar", "una-contraseña")

	if err == nil {
		t.Fatal("Register no devolvió error")
	}
	if len(userRepository.inserted) != 0 {
		t.Error("se creó una cuenta sin hash de contraseña")
	}
}

// seedAccount deja una cuenta lista para iniciar sesión.
func seedAccount(t *testing.T, auth *service.Auth, email, password string) model.User {
	t.Helper()

	user, err := auth.Register(t.Context(), email, password)
	if err != nil {
		t.Fatalf("no se pudo sembrar la cuenta: %v", err)
	}

	return user
}

func TestLoginIssuesASessionForCorrectCredentials(t *testing.T) {
	auth, _, _, tokenIssuer := newAuthServiceWithIssuer()
	user := seedAccount(t, auth, "nico@nosepudo.ar", "una-contraseña")

	session, err := auth.Login(t.Context(), "nico@nosepudo.ar", "una-contraseña")
	if err != nil {
		t.Fatalf("Login devolvió error: %v", err)
	}

	if session.AccessToken == "" {
		t.Error("no se emitió credencial de acceso")
	}
	if !session.AccessExpiresAt.Equal(tokenIssuer.expiresAt) {
		t.Errorf("AccessExpiresAt = %v, se esperaba %v", session.AccessExpiresAt, tokenIssuer.expiresAt)
	}

	// El actor y el privilegio salen de la cuenta, nunca de lo que mandó el cliente.
	if tokenIssuer.gotSubject != user.ID {
		t.Errorf("se emitió para la cuenta %d, se esperaba %d", tokenIssuer.gotSubject, user.ID)
	}
	if tokenIssuer.gotPrivilege != model.PrivilegeUser {
		t.Errorf("se emitió con privilegio %v", tokenIssuer.gotPrivilege)
	}
}

// La tabla de decisión de FR-003: sólo la fila en que la cuenta existe y la
// contraseña coincide emite algo, y las otras dos son indistinguibles entre sí.
func TestLoginDecisionTable(t *testing.T) {
	cases := []struct {
		name        string
		email       string
		password    string
		wantSession bool
	}{
		{"la cuenta existe y la contraseña coincide", "nico@nosepudo.ar", "una-contraseña", true},
		{"la cuenta existe y la contraseña no coincide", "nico@nosepudo.ar", "otra-contraseña", false},
		{"la cuenta no existe", "nadie@nosepudo.ar", "una-contraseña", false},
		{"la cuenta no existe y la contraseña tampoco", "nadie@nosepudo.ar", "cualquiera", false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			auth, _, _, tokenIssuer := newAuthServiceWithIssuer()
			seedAccount(t, auth, "nico@nosepudo.ar", "una-contraseña")
			issuedBefore := tokenIssuer.issued

			session, err := auth.Login(t.Context(), c.email, c.password)

			if c.wantSession {
				if err != nil {
					t.Fatalf("Login devolvió error: %v", err)
				}
				if session.AccessToken == "" {
					t.Error("no se emitió credencial")
				}
				return
			}

			if !errors.Is(err, model.ErrInvalidCredentials) {
				t.Errorf("Login devolvió %v, se esperaba ErrInvalidCredentials", err)
			}
			if session.AccessToken != "" {
				t.Error("se emitió una credencial pese al fallo")
			}
			if tokenIssuer.issued != issuedBefore {
				t.Error("se pidió una credencial pese al fallo")
			}
		})
	}
}

// El camino de "la cuenta no existe" paga igual el costo del hashing, o la
// duración de la respuesta delataría qué direcciones están registradas.
func TestLoginPaysTheHashingCostForAnUnknownAccount(t *testing.T) {
	auth, _, passwordHasher, _ := newAuthServiceWithIssuer()
	seedAccount(t, auth, "nico@nosepudo.ar", "una-contraseña")

	if _, err := auth.Login(t.Context(), "nadie@nosepudo.ar", "una-contraseña"); err == nil {
		t.Fatal("Login aceptó una cuenta inexistente")
	}

	if passwordHasher.dummyCompares != 1 {
		t.Errorf("se hicieron %d comparaciones señuelo, se esperaba 1", passwordHasher.dummyCompares)
	}
}

// El mismo caso borde de la spec, ahora en el login: iniciar sesión con otra
// forma de la dirección encuentra la misma cuenta.
func TestLoginNormalizesTheIdentifier(t *testing.T) {
	auth, _, _, _ := newAuthServiceWithIssuer()
	user := seedAccount(t, auth, "nico@nosepudo.ar", "una-contraseña")

	session, err := auth.Login(t.Context(), "  NICO@NoSePudo.AR  ", "una-contraseña")
	if err != nil {
		t.Fatalf("Login devolvió error: %v", err)
	}
	if session.AccessToken == "" {
		t.Errorf("no se emitió credencial para la cuenta %d", user.ID)
	}
}

// Dos inicios de sesión de la misma cuenta abren familias distintas: es lo que
// después va a permitir cerrar una sin cerrar la otra.
func TestLoginOpensAFreshSessionFamilyEachTime(t *testing.T) {
	auth, _, _, tokenIssuer := newAuthServiceWithIssuer()
	seedAccount(t, auth, "nico@nosepudo.ar", "una-contraseña")

	for range 2 {
		if _, err := auth.Login(t.Context(), "nico@nosepudo.ar", "una-contraseña"); err != nil {
			t.Fatalf("Login devolvió error: %v", err)
		}
	}

	if len(tokenIssuer.sessionIDs) != 2 {
		t.Fatalf("se emitieron %d credenciales, se esperaban 2", len(tokenIssuer.sessionIDs))
	}
	for i, sessionID := range tokenIssuer.sessionIDs {
		if sessionID == "" {
			t.Errorf("la credencial %d no lleva familia de sesión", i)
		}
	}
	if tokenIssuer.sessionIDs[0] == tokenIssuer.sessionIDs[1] {
		t.Error("los dos inicios de sesión comparten familia: cerrar uno cerraría el otro")
	}
}

// Un fallo de la base al buscar la cuenta no puede confundirse con
// credenciales inválidas: son un 500 y un 401, y no dicen lo mismo.
func TestLoginDistinguishesALookupFailureFromBadCredentials(t *testing.T) {
	auth, userRepository, _, _ := newAuthServiceWithIssuer()
	lookupFailure := errors.New("la base no responde")
	userRepository.getByEmailer = func(string) (model.User, error) {
		return model.User{}, lookupFailure
	}

	_, err := auth.Login(t.Context(), "nico@nosepudo.ar", "una-contraseña")

	if !errors.Is(err, lookupFailure) {
		t.Errorf("Login devolvió %v, se esperaba que propagara el fallo", err)
	}
	if errors.Is(err, model.ErrInvalidCredentials) {
		t.Error("un fallo de la base se reportó como credenciales inválidas")
	}
}

func TestLoginReturnsNoSessionWhenIssuingFails(t *testing.T) {
	auth, _, _, tokenIssuer := newAuthServiceWithIssuer()
	seedAccount(t, auth, "nico@nosepudo.ar", "una-contraseña")
	tokenIssuer.err = errors.New("no se pudo firmar")

	session, err := auth.Login(t.Context(), "nico@nosepudo.ar", "una-contraseña")

	if err == nil {
		t.Fatal("Login no devolvió error")
	}
	if session.AccessToken != "" {
		t.Error("se devolvió una sesión pese a que la firma falló")
	}
}
