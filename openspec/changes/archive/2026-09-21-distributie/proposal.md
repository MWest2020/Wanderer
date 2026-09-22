# Proposal: iemand moet het binnen een minuut kunnen proberen

## Why

Er hangen **nul bestanden** aan de releases van deze repo. Wie erover
leest, moet Go installeren, de repo klonen en zelf bouwen voordat hij
één scan ziet. Dat is de hele drempel tussen "interessant" en "ik
probeer het".

En er is een tweede kans die nu blijft liggen: dit is bij uitstek een
controle die je in een pijplijn wilt hebben, zoals je linters en tests
daar hebt. Iemand die één regel in zijn workflow zet en bij elke
wijziging ziet hoe zijn eigen domein ervoor staat, heeft de tool in zijn
eigen werk staan in plaats van op een website die hij moet bezoeken.

## What Changes

**Binaries bij elke release.** `goreleaser` bouwt bij een tag voor
Linux, macOS en Windows (amd64 en arm64), met checksums, en hangt ze aan
de GitHub-release. `CGO_ENABLED=0` kan hier: de database is
`modernc.org/sqlite`, pure Go, dus er is niets te linken.

**Eén regel om het te proberen.** De README opent met een
`curl`-regel die de binary haalt, en een `docker run`-regel voor wie
dat liever doet. Beide worden gecontroleerd door ze in CI te draaien —
een quickstart die niemand test, is een quickstart die stukgaat.

**Een GitHub Action.** Een action die een domein scant en het oordeel
als job-samenvatting toont: het aantal beantwoorde vragen, wat het
oordeel bepaalde, en de handeling. Bij een pull request kan dat als
commentaar. De action faalt niet standaard op een "nee" — dat is een
keuze van de gebruiker (`fail-on: afhankelijk`), want een soevereiniteitsoordeel
hoort niet zomaar iemands pijplijn rood te maken.

## Scope / Not in scope

**In:** goreleaser, de release-workflow, de quickstart met controle, de
action plus een voorbeeldworkflow en documentatie.

**Out:** publiceren in pakketbeheerders (brew, apt), en een aparte
CLI-image naast het bestaande exapp-image. Ook out: de action laten
scannen vanuit onze eigen instantie — hij draait bij de gebruiker, op
zijn eigen domeinen, precies zoals de scanner bedoeld is.

## Wat dit eerlijk houdt

Een action die stilletjes iemands domeinen naar ons stuurt, zou een
meetinstrument in een dataverzamelaar veranderen. Deze action doet dat
niet: hij draait in de runner van de gebruiker, praat alleen met het
domein dat hij meet, en stuurt niets naar ons. Dat staat in de
documentatie, en het is te controleren omdat de action niets anders is
dan de binary.
