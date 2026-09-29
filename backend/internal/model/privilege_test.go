package model_test

import (
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

func TestPrivilegeStringRoundTripsThroughParse(t *testing.T) {
	for _, level := range []model.PrivilegeLevel{model.PrivilegeUser, model.PrivilegeSuperuser} {
		if got := model.ParsePrivilege(level.String()); got != level {
			t.Errorf("ParsePrivilege(%q) = %v, se esperaba %v", level.String(), got, level)
		}
	}
}

func TestParsePrivilegeUnrecognizedIsUnknown(t *testing.T) {
	cases := []string{"", "User", "SUPERUSER", "admin", "root", "0", "1"}

	for _, s := range cases {
		t.Run(s, func(t *testing.T) {
			if got := model.ParsePrivilege(s); got != model.PrivilegeUnknown {
				t.Errorf("ParsePrivilege(%q) = %v, se esperaba PrivilegeUnknown", s, got)
			}
		})
	}
}

func TestPrivilegeUnknownRendersAsUnknown(t *testing.T) {
	if got := model.PrivilegeUnknown.String(); got != "unknown" {
		t.Errorf("PrivilegeUnknown.String() = %q, se esperaba \"unknown\"", got)
	}
}

func TestPrivilegeSatisfiesDecisionTable(t *testing.T) {
	cases := []struct {
		name string
		held model.PrivilegeLevel
		required model.PrivilegeLevel
		want bool
	}{
		{"usuario alcanza para usuario", model.PrivilegeUser, model.PrivilegeUser, true},
		{"superusuario alcanza para usuario", model.PrivilegeSuperuser, model.PrivilegeUser, true},
		{"superusuario alcanza para superusuario", model.PrivilegeSuperuser, model.PrivilegeSuperuser, true},
		{"usuario no alcanza para superusuario", model.PrivilegeUser, model.PrivilegeSuperuser, false},
		{"desconocido no alcanza para usuario", model.PrivilegeUnknown, model.PrivilegeUser, false},
		{"desconocido no alcanza para superusuario", model.PrivilegeUnknown, model.PrivilegeSuperuser, false},
		{"usuario no satisface un requisito desconocido", model.PrivilegeUser, model.PrivilegeUnknown, false},
		{"superusuario no satisface un requisito desconocido", model.PrivilegeSuperuser, model.PrivilegeUnknown, false},
		{"desconocido no satisface desconocido", model.PrivilegeUnknown, model.PrivilegeUnknown, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.held.Satisfies(c.required); got != c.want {
				t.Errorf("%v.Satisfies(%v) = %v, se esperaba %v", c.held, c.required, got, c.want)
			}
		})
	}
}

func TestPrivilegeZeroValueIsUnknown(t *testing.T) {
	var zero model.PrivilegeLevel

	if zero != model.PrivilegeUnknown {
		t.Fatalf("el valor cero es %v, se esperaba PrivilegeUnknown", zero)
	}
	if zero.Satisfies(model.PrivilegeUser) {
		t.Error("el valor cero satisface PrivilegeUser: un campo olvidado daría acceso")
	}
}

