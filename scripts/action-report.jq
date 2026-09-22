# Collapses a `wanderer assess --format json` Assessment into the
# shape action-report.sh needs for the job summary: an overall verdict
# ("worst wins" across dimensions, same rule docs/reference/assessor.md
# documents for a single dimension), the answered/total question count,
# which rule(s) decided the verdict, and every afhankelijk rule
# (the failing points the action reports one handeling for each).
def score_rank: {"afhankelijk": 1, "voldoende": 2, "soeverein": 3}[.] // null;

. as $assessment
| ($assessment.dimensions // []) as $dims
| ($dims | map(select(.not_applicable != true and .score != "onbekend"))) as $scored
| (if ($scored | length) > 0 then ($scored | map(.score | score_rank) | min) else null end) as $worst_rank
| (if $worst_rank == null then "onbekend" else ($scored | map(select((.score | score_rank) == $worst_rank)) | .[0].score) end) as $verdict
| ($dims | [.[].rationale[]?]) as $all_rationale
| ($all_rationale | map(select((.evidence // []) | length > 0))) as $evidenced
| {
    verdict: $verdict,
    answered: ($evidenced | length),
    total: ($all_rationale | length),
    score: "\($evidenced | length)/\($all_rationale | length)",
    scored_dimensions: ($scored | length),
    total_dimensions: ($dims | length),
    deciding: (
      if $worst_rank == null then []
      else
        [ $dims[]
          | select(.not_applicable != true and .score == $verdict)
          | .dimension as $dimension
          | .rationale[]?
          | select(.score == $verdict and ((.evidence // []) | length > 0))
          | {dimension: $dimension, criterium_id: .criterium_id, verdict: .verdict}
        ]
      end
    ),
    failing: [
      $dims[] | .dimension as $dimension | .rationale[]?
      | select(.score == "afhankelijk")
      | {dimension: $dimension, criterium_id: .criterium_id, verdict: .verdict, handeling: (.handeling // null)}
    ],
    dimensions: [
      $dims[] | {dimension: .dimension, score: .score, completeness: .completeness, not_applicable: (.not_applicable // false)}
    ]
  }
