package repository_test

import (
	"errors"
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/dao"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/repository"
)

func newUserRepository(t *testing.T) *repository.UserRepository {
	t.Helper()
	return repository.NewUserRepository(dao.NewUserDao(startPostgres(t)))
}

// fakeBcryptHash tiene la forma y el largo exactos de un digest de bcrypt:
// el prefijo de 7, 22 de salt y 31 de hash, 60 en total, que es justo lo que
// acepta la columna password_hash. No se hashea de verdad porque lo que estos
// casos prueban es el SQL y no el hashing.
const fakeBcryptHash = "$2a$04$" + "abcdefghijklmnopqrstuv" + "wxyz0123456789ABCDEFGHIJKLMNOPQ"

// Si alguien toca el fixture y se pasa de largo, conviene que falle acá y no
// tres casos más abajo con un error del driver.
func TestFixtureHashMatchesTheColumnWidth(t *testing.T) {
	if len(fakeBcryptHash) != 60 {
		t.Fatalf("el hash de prueba mide %d caracteres y la columna acepta 60", len(fakeBcryptHash))
	}
}

func aUser(email string) model.User {
	return model.User{
		Email: email,
		PasswordHash: fakeBcryptHash,
		Privilege: model.PrivilegeUser,
		Active: true,
	}
}

func TestUserRepositoryInsertAssignsAnIdentifierAndReadsBack(t *testing.T) {
	repo := newUserRepository(t)
	ctx := t.Context()

	inserted, err := repo.Insert(ctx, aUser("nico@nosepudo.ar"))
	if err != nil {
		t.Fatalf("Insert devolvió error: %v", err)
	}

	if inserted.ID == 0 {
		t.Error("la base no asignó un identificador")
	}
	if inserted.CreatedAt.IsZero() {
		t.Error("la base no asignó created_at")
	}

	found, err := repo.GetByEmail(ctx, "nico@nosepudo.ar")
	if err != nil {
		t.Fatalf("GetByEmail devolvió error: %v", err)
	}

	if found.ID != inserted.ID {
		t.Errorf("ID = %d, se esperaba %d", found.ID, inserted.ID)
	}
	if found.Email != "nico@nosepudo.ar" {
		t.Errorf("Email = %q", found.Email)
	}
	if found.PasswordHash != inserted.PasswordHash {
		t.Errorf("el hash guardado no es el que se insertó")
	}
	if found.Privilege != model.PrivilegeUser {
		t.Errorf("Privilege = %v, se esperaba user", found.Privilege)
	}
	if !found.Active {
		t.Error("Active = false, se esperaba true")
	}
}

// el requerimiento: es el índice único el que decide, no un chequeo previo del service.
func TestUserRepositoryInsertRefusesADuplicateEmail(t *testing.T) {
	repo := newUserRepository(t)
	ctx := t.Context()

	first, err := repo.Insert(ctx, aUser("nico@nosepudo.ar"))
	if err != nil {
		t.Fatalf("la primera inserción devolvió error: %v", err)
	}

	_, err = repo.Insert(ctx, aUser("nico@nosepudo.ar"))
	if !errors.Is(err, model.ErrEmailTaken) {
		t.Fatalf("la segunda inserción devolvió %v, se esperaba ErrEmailTaken", err)
	}

	// La cuenta existente queda intacta.
	found, err := repo.GetByEmail(ctx, "nico@nosepudo.ar")
	if err != nil {
		t.Fatalf("GetByEmail devolvió error: %v", err)
	}
	if found.ID != first.ID {
		t.Errorf("la cuenta existente cambió: ID = %d, se esperaba %d", found.ID, first.ID)
	}
}

func TestUserRepositoryGetByEmailReportsAMiss(t *testing.T) {
	repo := newUserRepository(t)

	_, err := repo.GetByEmail(t.Context(), "nadie@nosepudo.ar")
	if !errors.Is(err, model.ErrUserNotFound) {
		t.Errorf("GetByEmail devolvió %v, se esperaba ErrUserNotFound", err)
	}
}

func TestUserRepositoryGetByIDReportsAMiss(t *testing.T) {
	repo := newUserRepository(t)

	_, err := repo.GetByID(t.Context(), 99999)
	if !errors.Is(err, model.ErrUserNotFound) {
		t.Errorf("GetByID devolvió %v, se esperaba ErrUserNotFound", err)
	}
}

func TestUserRepositoryGetByIDFindsTheAccount(t *testing.T) {
	repo := newUserRepository(t)
	ctx := t.Context()

	inserted, err := repo.Insert(ctx, aUser("nico@nosepudo.ar"))
	if err != nil {
		t.Fatalf("Insert devolvió error: %v", err)
	}

	found, err := repo.GetByID(ctx, inserted.ID)
	if err != nil {
		t.Fatalf("GetByID devolvió error: %v", err)
	}
	if found.Email != inserted.Email {
		t.Errorf("Email = %q, se esperaba %q", found.Email, inserted.Email)
	}
}

// El nivel de privilegio es una propiedad de la cuenta y tiene que sobrevivir
// la ida y vuelta a la base: es lo que el credencial va a declarar después.
func TestUserRepositoryPreservesTheSuperuserPrivilege(t *testing.T) {
	repo := newUserRepository(t)
	ctx := t.Context()

	superuser := aUser("admin@nosepudo.ar")
	superuser.Privilege = model.PrivilegeSuperuser

	if _, err := repo.Insert(ctx, superuser); err != nil {
		t.Fatalf("Insert devolvió error: %v", err)
	}

	found, err := repo.GetByEmail(ctx, "admin@nosepudo.ar")
	if err != nil {
		t.Fatalf("GetByEmail devolvió error: %v", err)
	}
	if found.Privilege != model.PrivilegeSuperuser {
		t.Errorf("Privilege = %v, se esperaba superuser", found.Privilege)
	}
}

// La columna guarda valores ya normalizados, así que dos formas de la misma
// dirección chocan contra el índice único. Es lo que cierra el caso borde de
// el diseño sobre mayúsculas y espacios.
func TestUserRepositoryNormalizedEmailsCollide(t *testing.T) {
	repo := newUserRepository(t)
	ctx := t.Context()

	if _, err := repo.Insert(ctx, aUser(model.NormalizeEmail(" Nico@NoSePudo.AR "))); err != nil {
		t.Fatalf("la primera inserción devolvió error: %v", err)
	}

	_, err := repo.Insert(ctx, aUser(model.NormalizeEmail("nico@nosepudo.ar")))
	if !errors.Is(err, model.ErrEmailTaken) {
		t.Errorf("Insert devolvió %v, se esperaba ErrEmailTaken: una dirección se volvió dos cuentas", err)
	}
}

