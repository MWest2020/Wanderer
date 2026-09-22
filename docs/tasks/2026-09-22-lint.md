# Habitat run — de elf resterende lint-meldingen

De lint-stap van CI staat rood. misspell en gofumpt zijn al opgelost
(commit "lint: misspell kent geen Nederlands"). Wat overblijft zijn elf
meldingen die een oordeel vragen. Draai zelf
`golangci-lint run ./...` — die staat in het image.

## Scope

### 1. G404 — dit is geen ruis, dit is een bevinding
`internal/probe/dns/resolver.go:136` en `internal/probe/soa/resolver.go:118`
maken het DNS-transactie-ID met `rand.Intn(1 << 16)` uit `math/rand`.

Dat ID is de enige bescherming tegen een vervalst antwoord: wie het kan
voorspellen, kan onze resolver een ander antwoord voeren dan de
nameserver gaf (RFC 5452 gaat hierover). Voor een scanner die
CAA-records en SOA-gegevens als bewijs in een rapport zet, is dat
precies het verkeerde onderdeel om zwak te laten.

Zet beide om naar `crypto/rand`. Onderdruk deze melding NIET.
Schrijf een test die vastlegt wat je wilt: bijvoorbeeld dat twee
opeenvolgende queries verschillende ID's hebben en dat het ID niet uit
een voorspelbare reeks komt. Zeg in je rapport hoe je hebt gecontroleerd
dat de test faalt zonder de reparatie.

### 2. G115 in tests (3x)
`internal/probe/dns/resolver_test.go:30,53,63` en
`internal/probe/soa/resolver_test.go:57,66`: `int -> uint16` in
testcode, waar de waarde uit de test zelf komt. Onderdruk per regel of
via een `issues.exclude-rules`-regel op `_test\.go` + `G115`, met een
comment dat zegt waarom dat hier veilig is (de waarde is een
literal/loop-index uit de test, geen invoer).

### 3. G204 in `scripts/action_report_test.go:26`
De test start een subproces met een pad dat uit de test komt. Zelfde
aanpak, zelfde eis: een comment die zegt waarom.

### 4. revive redefines-builtin-id (4x) + unused-parameter (1x)
- `internal/probe/http/http.go:465,481` — een type dat `any` heet.
- `internal/assessor/wand/rules_test.go:306,310` — een functie `min`.
- `internal/agent/enrol_test.go:76` — parameter `w` ongebruikt.
Hernoem. Kies namen die zeggen wat het ding is, niet `any2`.

## Done means
`golangci-lint run ./...` geeft NUL meldingen, en
`go build ./...`, `go vet ./...`, `go test ./...` zijn groen.
Budget is $6.
