package models_test

import (
	"strings"
	"testing"

	"github.com/MWest2020/wanderer/pkg/models"
)

func TestOrganisation_Validate_AcceptsCanonical(t *testing.T) {
	o := &models.Organisation{Slug: "acme", Name: "ACME B.V."}
	if err := o.Validate(); err != nil {
		t.Fatalf("canonical slug rejected: %v", err)
	}
}

func TestOrganisation_Validate_RejectsEmptySlug(t *testing.T) {
	o := &models.Organisation{Name: "ACME"}
	if err := o.Validate(); err == nil {
		t.Fatal("empty slug must be rejected")
	}
}

func TestOrganisation_Validate_RejectsLeadingHyphen(t *testing.T) {
	o := &models.Organisation{Slug: "-acme", Name: "ACME"}
	if err := o.Validate(); err == nil {
		t.Fatal("leading hyphen must be rejected")
	}
}

func TestOrganisation_Validate_RejectsTrailingHyphen(t *testing.T) {
	o := &models.Organisation{Slug: "acme-", Name: "ACME"}
	if err := o.Validate(); err == nil {
		t.Fatal("trailing hyphen must be rejected")
	}
}

func TestOrganisation_Validate_RejectsUppercase(t *testing.T) {
	o := &models.Organisation{Slug: "ACME", Name: "ACME"}
	if err := o.Validate(); err == nil {
		t.Fatal("uppercase letters must be rejected")
	}
}

func TestOrganisation_Validate_RejectsTooShort(t *testing.T) {
	o := &models.Organisation{Slug: "a", Name: "A"}
	if err := o.Validate(); err == nil {
		t.Fatal("single-char slug must be rejected (min 2)")
	}
}

func TestOrganisation_Validate_RejectsTooLong(t *testing.T) {
	o := &models.Organisation{Slug: strings.Repeat("a", 41), Name: "A"}
	if err := o.Validate(); err == nil {
		t.Fatal("41-char slug must be rejected (max 40)")
	}
}

func TestOrganisation_Validate_AcceptsHyphenInMiddle(t *testing.T) {
	o := &models.Organisation{Slug: "ac-me", Name: "AC-ME"}
	if err := o.Validate(); err != nil {
		t.Fatalf("middle hyphen rejected: %v", err)
	}
}

func TestOrganisation_Validate_AcceptsDigits(t *testing.T) {
	o := &models.Organisation{Slug: "acme-2026", Name: "ACME 2026"}
	if err := o.Validate(); err != nil {
		t.Fatalf("digits rejected: %v", err)
	}
}

func TestOrganisation_Validate_RejectsEmptyName(t *testing.T) {
	o := &models.Organisation{Slug: "acme"}
	if err := o.Validate(); err == nil {
		t.Fatal("empty name must be rejected")
	}
}

func TestOrganisation_Validate_RejectsUnderscore(t *testing.T) {
	o := &models.Organisation{Slug: "ac_me", Name: "ACME"}
	if err := o.Validate(); err == nil {
		t.Fatal("underscore must be rejected")
	}
}
