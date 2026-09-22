package assessor

import "testing"

func TestReasonInfo_SeededCodes(t *testing.T) {
	cases := []struct {
		code    string
		class   ReasonClass
		subject ReasonSubject
	}{
		{ReasonRegistryRedacted, ReasonStructural, ReasonSubjectTarget},
		{ReasonNotPublishedByRegistry, ReasonStructural, ReasonSubjectTarget},
		{ReasonScannerNoIPv6, ReasonStructural, ReasonSubjectScanner},
		{ReasonProbeUnavailable, ReasonGap, ReasonSubjectTarget},
		{ReasonScannerUnreadableOutput, ReasonGap, ReasonSubjectScanner},
	}
	for _, c := range cases {
		t.Run(c.code, func(t *testing.T) {
			class, subject := ReasonInfo(c.code)
			if class != c.class {
				t.Errorf("class = %s, want %s", class, c.class)
			}
			if subject != c.subject {
				t.Errorf("subject = %s, want %s", subject, c.subject)
			}
		})
	}
}

func TestReasonInfo_UnregisteredCodePanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("want panic for an unregistered reason code")
		}
	}()
	ReasonInfo("totally_made_up")
}
