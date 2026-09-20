# Proposal: maak de Playwright-suite weer groen (strict-mode locators)

## Why

`make playwright` faalt op zes specs, en dat doet het al minstens sinds
`682807f` — dus ruim vóór de accountability-change. Gemeten op
2026-09-20 met Chromium 147: **31 geslaagd, 6 gefaald**, identiek op
`682807f` en op `main`.

Alle zes falen op dezelfde manier: een locator die meer dan één element
raakt.

```
locator('text=wand.juridisch.cert_issuer_eea') resolved to 2 elements
getByRole('heading', { name: /Verdict/i }) resolved to 2 elements
```

De oorzaak is niet een fout in de pagina, maar een verouderde
aanname in de test. `/ui/trends` toont sinds de personas-consolidatie
(ADR-0017) twee tabellen — `table.rule-catalogue` en
`table.reporting-rules` — die allebei regel-ID's tonen; `wand.juridisch.
cert_issuer_eea` staat er vier keer op de pagina. En op `/ui/` staan twee
koppen die op `/Verdict/i` matchen ("Targets · your fleet…" bevat het
woord niet, maar de regex raakt de tweede kop ook).

Een suite waarin zes specs altijd rood staan, is erger dan geen suite:
niemand kijkt nog naar de uitslag, en een échte regressie valt niet meer
op. Dat is precies wat hier gebeurde — de accountability-spec uit run 08
stond in geen enkele `testMatch` en viel daardoor niemand op.

## What Changes

- De zes specs krijgen locators die op één element wijzen: scope binnen
  de bedoelde tabel (`table.rule-catalogue`) of kop, in plaats van een
  kale `text=`/regex-match over de hele pagina.
- De suite draait weer groen: 37 van 37.
- `project-hygiene` krijgt de eis dat de Playwright-suite groen is en
  dat locators op één element wijzen.

## Scope / Not in scope

**In:** `tests/playwright/specs/{dar,reporting-catalogue,
container-image-sovereignty,eu-package-origin,host-side-scoring,
nextcloud-as-target}.spec.ts`, en de hygiëne-eis.

**Out:** de UI zelf. De pagina's kloppen; twee tabellen die allebei
regel-ID's tonen is een bewuste keuze van ADR-0017. Wie dat wil
veranderen, doet dat in een eigen change. Ook buiten scope: nieuwe
dekking toevoegen.

## Build process

Eén habitat-run met task-ref `runs/01-fix.md`. De kooi kan de
Playwright-browser niet downloaden (geen egress naar de Playwright-CDN),
dus de run levert de aanpassing en **ik draai de suite hier na** —
dat staat zo in de task-ref.
