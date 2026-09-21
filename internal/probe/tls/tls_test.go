package tls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"testing"
	"time"

	"github.com/MWest2020/wanderer/pkg/models"
)

// selfSignedCert builds a self-signed leaf certificate with a chosen
// NotAfter, so tests can drive inspectState's expiring_soon boundary
// without a real handshake.
func selfSignedCert(t *testing.T, notAfter time.Time) *x509.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "expiry.example"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     notAfter,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse certificate: %v", err)
	}
	return cert
}

func validityFinding(t *testing.T, findings []models.Finding) models.Finding {
	t.Helper()
	for _, f := range findings {
		if f.ProbeID == "tls.validity" {
			return f
		}
	}
	t.Fatalf("no tls.validity finding, got %+v", findings)
	return models.Finding{}
}

// TestInspectState_ExpiringSoonThreshold covers spec scenario
// "Certificaat verloopt binnenkort": the tls.validity finding always
// carries the threshold expiring_soon rests on, derived from
// expiringSoonThresholdDays rather than a literal in the test, so
// changing the constant without changing the comparison (or vice
// versa) fails here.
func TestInspectState_ExpiringSoonThreshold(t *testing.T) {
	cases := []struct {
		name             string
		daysUntilExpiry  int
		wantExpired      bool
		wantExpiringSoon bool
	}{
		{"well beyond the threshold", expiringSoonThresholdDays + 10, false, false},
		{"just inside the threshold", expiringSoonThresholdDays - 10, false, true},
		{"already expired", -1, true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			notAfter := time.Now().Add(time.Duration(tc.daysUntilExpiry) * 24 * time.Hour)
			cert := selfSignedCert(t, notAfter)
			state := &tls.ConnectionState{PeerCertificates: []*x509.Certificate{cert}}
			f := validityFinding(t, inspectState("expiry.example", state))

			expired, _ := f.Attributes["expired"].(bool)
			if expired != tc.wantExpired {
				t.Errorf("expired = %v, want %v", expired, tc.wantExpired)
			}
			expiringSoon, _ := f.Attributes["expiring_soon"].(bool)
			if expiringSoon != tc.wantExpiringSoon {
				t.Errorf("expiring_soon = %v, want %v", expiringSoon, tc.wantExpiringSoon)
			}

			threshold, ok := f.Attributes["expiring_soon_threshold"].(int)
			if !ok || threshold != expiringSoonThresholdDays {
				t.Errorf("expiring_soon_threshold = %v (ok=%v), want %d", f.Attributes["expiring_soon_threshold"], ok, expiringSoonThresholdDays)
			}
			unit, _ := f.Attributes["expiring_soon_threshold_unit"].(string)
			if unit != "days" {
				t.Errorf("expiring_soon_threshold_unit = %q, want %q", unit, "days")
			}
		})
	}
}

// TestInspectState_ExpiringSoonFlagMatchesThreshold covers spec
// scenario "Vlag en grens lopen uiteen": right at the boundary the
// constant defines, the flag and the delivered number must agree —
// exercising the exact comparison expiringSoonThresholdDays drives, so
// a divergence between the comparison and the delivered number fails
// this test.
func TestInspectState_ExpiringSoonFlagMatchesThreshold(t *testing.T) {
	notAfter := time.Now().Add(time.Duration(expiringSoonThresholdDays)*24*time.Hour - time.Minute)
	cert := selfSignedCert(t, notAfter)
	state := &tls.ConnectionState{PeerCertificates: []*x509.Certificate{cert}}
	f := validityFinding(t, inspectState("expiry.example", state))

	expiringSoon, _ := f.Attributes["expiring_soon"].(bool)
	threshold, _ := f.Attributes["expiring_soon_threshold"].(int)
	if !expiringSoon {
		t.Fatalf("expiring_soon = false just inside the %d-day threshold", expiringSoonThresholdDays)
	}
	if threshold != expiringSoonThresholdDays {
		t.Fatalf("expiring_soon fired using a boundary of %d days, but delivered threshold %d — flag and number disagree", expiringSoonThresholdDays, threshold)
	}
}
