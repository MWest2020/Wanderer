# Proposal: een geplande scan levert ook een oordeel

## Why

De scheduler draait scans, maar produceert geen `Assessment`. Op de
live-instantie (wanderer.westerweel.work, 2026-09-20) werd dat meteen
zichtbaar: een scan via `POST /scans` gaf 71 findings, en de
rapportpagina zei

> No assessment has been produced for this scan yet. Run `wanderer
> assess <id> --framework both` …

Pas na een handmatige `wanderer assess` in de pod verscheen de
antwoordlijst. Voor een instantie die op een schema draait en door
mensen gelezen wordt, is dat een dood spoor: de scan is er, het oordeel
niet, en de bezoeker krijgt een instructie voor een commando dat hij
niet kan draaien.

Findings zonder oordeel zijn ook geen half product maar een verkeerd
product: de hele opzet van Wanderer is dat scoring uit findings volgt.

## What Changes

- Na elke geslaagde geplande scan beoordeelt de scheduler die scan met
  dezelfde code als `wanderer assess --framework both`, en slaat het
  oordeel op.
- Faalt het beoordelen, dan blijft de scan bestaan en logt de scheduler
  de fout — een kapotte regel mag geen scans laten verdwijnen.
- Een schakelaar in de schedules-configuratie (`assess: false`) zet het
  per schedule uit; standaard staat het aan.

## Scope / Not in scope

**In:** `internal/scheduler` en de configuratie ervan.

**Out:** `POST /scans` (de API-route blijft zoals hij is — daar hoort
`POST /scans/{id}/assessments` bij), de CLI, en het herbeoordelen van
oude scans.

## Build process

Eén habitat-run met task-ref `runs/01-assess.md`.
