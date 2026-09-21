package domainutil

import "testing"

func TestRegistrable(t *testing.T) {
	cases := []struct {
		host string
		want string
	}{
		{"ns1.provider-a.nl", "provider-a.nl"},
		{"ns2.provider-a.nl.", "provider-a.nl"},
		{"ns1.provider-b.eu", "provider-b.eu"},
		{"provider-a.nl", "provider-a.nl"},
		{"nl", ""},
		{"", ""},
	}
	for _, c := range cases {
		if got := Registrable(c.host); got != c.want {
			t.Errorf("Registrable(%q) = %q, want %q", c.host, got, c.want)
		}
	}
}
