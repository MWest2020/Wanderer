package assessor_test

import (
	"testing"

	"github.com/MWest2020/wanderer/internal/assessor"
	"github.com/MWest2020/wanderer/pkg/models"
)

func TestIsEvidenceLike(t *testing.T) {
	cases := []struct {
		name string
		f    models.Finding
		want bool
	}{
		{
			name: "real evidence (no meta attributes)",
			f:    models.Finding{ProbeID: "dns.mx", Attributes: map[string]any{"host": "mail.example.nl"}},
			want: true,
		},
		{
			name: "nil attributes",
			f:    models.Finding{ProbeID: "dns.mx"},
			want: true,
		},
		{
			name: "lookup error",
			f:    models.Finding{ProbeID: "dns.mx", Attributes: map[string]any{"error": "no such host", "kind": "nxdomain"}},
			want: false,
		},
		{
			name: "no_answer true",
			f:    models.Finding{ProbeID: "dns.caa", Attributes: map[string]any{"no_answer": true, "reason": "no CAA records"}},
			want: false,
		},
		{
			name: "unavailable true",
			f:    models.Finding{ProbeID: "ip.unavailable", Attributes: map[string]any{"unavailable": true, "reason": "no GeoLite2 DB"}},
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := assessor.IsEvidenceLike(tc.f); got != tc.want {
				t.Errorf("IsEvidenceLike = %v, want %v", got, tc.want)
			}
		})
	}
}
