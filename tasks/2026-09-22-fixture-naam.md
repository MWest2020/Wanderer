> Uitgevoerd 2026-09-22. De naam zelf staat hier niet meer in; zie
> commit b28dbaf en voorgangers voor wat er verving.

# Habitat run — de naam van de vorige eigenaar uit de testdata

Mark, 2026-09-22: die naam moet overal weg. De
eigenaarsvermeldingen in docs, README, ADR-0011, action.yml en de
specs zijn al aangepast. Wat overblijft is testdata — en die is wél
zichtbaar: de baseline-fixture rendert in de dev- en test-UI, en drie
Playwright-specs verwachten de naam letterlijk op het scherm.

## Waar het staat

- `internal/fixtures/baseline.go`: de organisatie-slug, naam en het doeldomein van de vorige
  eigenaar, plus een `whois.registrant`-finding met diezelfde naam,
  en drie comments bovenin die het scenario beschrijven.
- `internal/store/organisation_test.go`: twee keer als hernoemdoel.
- `tests/playwright/specs/dar.spec.ts`: drie asserties op de naam in
  de `h1` en de `.scope-pill`.

## Wat het moet worden

Volg de stijl die er al is: de tweede organisatie heet `acme` /
`ACME B.V.` met `acme.example.com`, en het tweede domein heet
`tweede.nl`. Gebruik dus een even herkenbare plaatshouder:

    slug          voorbeeld
    naam          Voorbeeld B.V.
    hoofddomein   voorbeeld.nl

Het scenario zelf blijft ongewijzigd: die organisatie is de volledig
soevereine (NL-uitgegeven TLS, NL-hosting) waar de vlootsortering en
de regelpagina-spec op leunen. Alleen de naam verandert.

## Scope — ONLY this
- [x] 1.1 De fixture, inclusief de comments die het scenario
  beschrijven.
- [x] 1.2 `organisation_test.go`.
- [x] 1.3 De drie asserties in `dar.spec.ts`.
- [x] 1.4 Zoek de hele repo na op de oude naam (buiten `vendor/`,
  `openspec/changes/archive/` en historische CHANGELOG-regels — die
  beschrijven wat er toen gebeurde en blijven staan). Meld in je
  rapport wat je nog vond.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` groen. Playwright
kun je niet draaien; zeg dat, ik meet na. Budget is $5.
