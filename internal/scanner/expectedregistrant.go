package scanner

import "github.com/MWest2020/wanderer/pkg/models"

// expectedRegistrantFinding builds the config.expected_registrant
// Finding that records, at scan time, which registrant names the
// scan's organisation declared (`wanderer org add
// --expected-registrant`). Recording the declared list as a Finding —
// rather than having the assessor read the organisations table
// directly — keeps the assessor a pure function of Findings (assessor
// spec: "the same Findings produce the same Assessment") and means a
// later re-assessment of this scan sees the names as they stood when
// the scan ran, not what the organisation says today. Always emitted,
// even when names is empty, so its absence is never mistaken for "not
// scanned yet".
func expectedRegistrantFinding(target models.Target, names []string) models.Finding {
	if names == nil {
		names = []string{}
	}
	return models.Finding{
		ProbeID:       "config.expected_registrant",
		DimensionHint: models.DimensionAccountability,
		Subject:       target.Domain,
		Severity:      models.SeverityInfo,
		Attributes: map[string]any{
			"expected_registrant": names,
		},
	}
}
