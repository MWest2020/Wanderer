package scheduler

import (
	"context"
	"fmt"

	"github.com/MWest2020/wanderer/internal/assessor"
	"github.com/MWest2020/wanderer/internal/assessor/eucsf"
	"github.com/MWest2020/wanderer/internal/assessor/wand"
	"github.com/MWest2020/wanderer/pkg/models"
)

// assessFrameworks are the rule packs a scheduled scan is judged
// against — the same set `wanderer assess --framework both` runs.
var assessFrameworks = []models.Framework{models.FrameworkWand, models.FrameworkEUCSF}

// assessmentPersister is the minimal store surface assessScan needs.
// A narrow interface (rather than *store.Store directly) lets tests
// exercise the "assessment fails" path without a real store failure.
type assessmentPersister interface {
	CreateAssessment(ctx context.Context, a *models.Assessment) error
}

// assessScan judges scan against every framework in assessFrameworks
// and persists the resulting Assessments. It reuses the same
// assessor.Assess/RenderMarkdown pipeline as `wanderer assess
// --framework both` (cmd/wanderer/assess.go) and
// `POST /scans/{id}/assessments` (internal/api/api.go) instead of
// re-deriving scoring logic here. Returns the first error hit; the
// caller decides whether that should stop the scan from being kept
// (it should not — see internal/scheduler/scheduler.go).
func assessScan(ctx context.Context, st assessmentPersister, scan *models.Scan, subject string) error {
	for _, fw := range assessFrameworks {
		rules := rulesForFramework(fw)
		a := &models.Assessment{
			ScanID:     scan.ID,
			Framework:  string(fw),
			Dimensions: assessor.Assess(scan.Findings, rules),
		}
		var buf strBuf
		if err := assessor.RenderMarkdown(&buf, a, assessor.Rules(rules), subject); err != nil {
			return fmt.Errorf("render %s report: %w", fw, err)
		}
		a.Report = buf.String()
		if err := st.CreateAssessment(ctx, a); err != nil {
			return fmt.Errorf("persist %s assessment: %w", fw, err)
		}
	}
	return nil
}

// rulesForFramework dispatches to the right rule pack, mirroring
// rulesForFramework in cmd/wanderer/assess.go.
func rulesForFramework(fw models.Framework) []assessor.Rule {
	switch fw {
	case models.FrameworkEUCSF:
		return eucsf.DefaultRules()
	default:
		return wand.DefaultRules()
	}
}

// strBuf is a minimal io.Writer-backed string builder, matching the
// one in cmd/wanderer/assess.go and internal/api/api.go.
type strBuf struct{ data []byte }

func (b *strBuf) Write(p []byte) (int, error) {
	b.data = append(b.data, p...)
	return len(p), nil
}
func (b *strBuf) String() string { return string(b.data) }
