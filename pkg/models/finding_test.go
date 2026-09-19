package models

import "testing"

func TestDimensionHint_Valid(t *testing.T) {
	cases := []struct {
		d    DimensionHint
		want bool
	}{
		{DimensionNone, true},
		{DimensionJuridisch, true},
		{DimensionTechnologie, true},
		{DimensionDataAI, true},
		{DimensionOperationeel, true},
		{DimensionMens, true},
		{DimensionAccountability, true},
		{DimensionHint("bogus"), false},
	}
	for _, c := range cases {
		if got := c.d.Valid(); got != c.want {
			t.Errorf("DimensionHint(%q).Valid() = %v, want %v", c.d, got, c.want)
		}
	}
}
