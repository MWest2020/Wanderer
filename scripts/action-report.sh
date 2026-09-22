#!/usr/bin/env bash
# Runs a scan against WANDERER_DOMAIN, assesses it, and writes the
# verdict to $GITHUB_STEP_SUMMARY plus the action's outputs. Used as
# the second step of action.yml. Everything it talks to is either the
# scanned domain itself or the local binary — see docs/how-to/action.md
# "What this action does not do".
set -euo pipefail

bin="${WANDERER_BIN:?WANDERER_BIN not set — run scripts/action-fetch.sh first}"
domain="${WANDERER_DOMAIN:?WANDERER_DOMAIN (action input 'domain') is required}"
geoip="${WANDERER_GEOIP:-}"
fail_on="${WANDERER_FAIL_ON:-}"
jq_filter="${WANDERER_JQ_FILTER:-$(dirname "${BASH_SOURCE[0]}")/action-report.jq}"

case "$fail_on" in
"" | "afhankelijk" | "onbekend") ;;
*)
	echo "wanderer-action: fail-on must be empty, 'afhankelijk', or 'onbekend' — got '$fail_on'" >&2
	exit 1
	;;
esac

work_dir="$(mktemp -d)"
db="$work_dir/wanderer.db"
report_path="$work_dir/assessment.json"

scan_args=(scan "$domain" --db "$db")
if [[ -n "$geoip" ]]; then
	scan_args+=(--geoip "$geoip")
fi

echo "wanderer-action: scanning $domain"
scan_output="$("$bin" "${scan_args[@]}")"
scan_id="$(head -n1 <<<"$scan_output" | awk '{print $2}')"
if [[ -z "$scan_id" ]]; then
	echo "wanderer-action: could not read a scan ID from scan output:" >&2
	echo "$scan_output" >&2
	exit 1
fi

echo "wanderer-action: assessing scan $scan_id"
"$bin" assess "$scan_id" --db "$db" --format json --persist=false >"$report_path"

summary_json="$(jq -f "$jq_filter" "$report_path")"
verdict="$(jq -r '.verdict' <<<"$summary_json")"
score="$(jq -r '.score' <<<"$summary_json")"
answered="$(jq -r '.answered' <<<"$summary_json")"
total="$(jq -r '.total' <<<"$summary_json")"
scored_dimensions="$(jq -r '.scored_dimensions' <<<"$summary_json")"
total_dimensions="$(jq -r '.total_dimensions' <<<"$summary_json")"

{
	echo "## Wanderer — soevereiniteitsscan van \`$domain\`"
	echo
	echo "**Score:** $score vragen beantwoord met bewijs ($answered van $total)"
	echo "**Oordeel:** \`$verdict\` (over $scored_dimensions van $total_dimensions dimensies met een score)"
	echo

	if [[ -z "$geoip" ]]; then
		echo "> ⚠️ Geen \`geoip\` opgegeven — jurisdictie-afhankelijke antwoorden (IP/ASN-land) zijn \`onbekend\`, niet meegewogen als \`voldoende\` of \`soeverein\`."
		echo
	fi

	echo "### Wat het oordeel bepaalde"
	deciding_count="$(jq -r '.deciding | length' <<<"$summary_json")"
	if [[ "$deciding_count" -eq 0 ]]; then
		echo "Geen enkele dimensie kon worden beoordeeld — alles staat op \`onbekend\`."
	else
		while IFS= read -r row; do
			dim="$(jq -r '.dimension' <<<"$row")"
			crit="$(jq -r '.criterium_id' <<<"$row")"
			verd="$(jq -r '.verdict' <<<"$row")"
			echo "- **$dim** — \`$crit\`: $verd"
		done < <(jq -c '.deciding[]' <<<"$summary_json")
	fi
	echo

	echo "### Handeling per falend punt"
	failing_count="$(jq -r '.failing | length' <<<"$summary_json")"
	if [[ "$failing_count" -eq 0 ]]; then
		echo "Geen falende punten (\`afhankelijk\`) gevonden."
	else
		while IFS= read -r row; do
			dim="$(jq -r '.dimension' <<<"$row")"
			crit="$(jq -r '.criterium_id' <<<"$row")"
			handeling="$(jq -r '.handeling // ""' <<<"$row")"
			if [[ -n "$handeling" ]]; then
				handeling="${handeling//\{domein\}/$domain}"
			else
				handeling="Geen kant-en-klare handeling beschikbaar voor deze regel — zie het volledige rapport."
			fi
			echo "- **\`$crit\`** ($dim): $handeling"
		done < <(jq -c '.failing[]' <<<"$summary_json")
	fi
	echo

	echo "### Per dimensie"
	echo "| Dimensie | Score | Volledigheid |"
	echo "| --- | --- | --- |"
	while IFS= read -r row; do
		dim="$(jq -r '.dimension' <<<"$row")"
		sc="$(jq -r '.score' <<<"$row")"
		comp="$(jq -r '.completeness' <<<"$row")"
		na="$(jq -r '.not_applicable' <<<"$row")"
		if [[ "$na" == "true" ]]; then
			comp="n.v.t."
		fi
		echo "| $dim | $sc | $comp |"
	done < <(jq -c '.dimensions[]' <<<"$summary_json")
	echo

	echo "Volledig rapport (JSON): \`$report_path\`"
} >>"$GITHUB_STEP_SUMMARY"

if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
	{
		echo "score=$score"
		echo "verdict=$verdict"
		echo "report-path=$report_path"
	} >>"$GITHUB_OUTPUT"
fi

echo "wanderer-action: verdict=$verdict score=$score report=$report_path"

if [[ -n "$fail_on" && "$verdict" == "$fail_on" ]]; then
	echo "wanderer-action: verdict '$verdict' matches fail-on '$fail_on' — failing the step" >&2
	exit 1
fi
