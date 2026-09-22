# Proposal: agents melden zichzelf aan, en de kern luistert ook echt

## Why

De agent bestaat en doet zijn werk: inventaris (systemd, pakketten,
Docker, Nextcloud) en egress (configbestanden, proces-omgeving,
systemd-units, en een eBPF-flowprobe), met een outbox op schijf die een
storing van dagen opvangt. Wat eromheen zit, is handwerk — en één stuk
ervan is stuk.

**1. De kern luistert niet.** `cmd/wanderer/serve.go` bouwt de router
met `api.Router(...)`, en die zet `AgentSecrets` op `nil`. De
documentatie zegt het zelf: met `nil` is de route geregistreerd maar
wordt *elk* verzoek geweigerd. Er bestaat een `RouterWithSecrets`, maar
niets in `serve` vult die. De remote-modus van de agent kan dus op een
standaard server niets afleveren.

**2. Aanmelden is handwerk met één sleutel.** Het vertrouwensmodel is
een gedeeld HMAC-geheim per hostnaam, dat je met de hand op beide
kanten neerlegt. Eén host erbij betekent de kern aanpassen; één host
die je niet meer vertrouwt, betekent hetzelfde. Er is geen manier om
één agent in te trekken zonder de rest te raken.

**3. Opnieuw afleveren kan dubbel tellen.** De outbox stuurt na een
storing alles alsnog. `AppendFindings` doet een kale `INSERT`, dus
dezelfde partij twee keer aangeboden levert dezelfde waarnemingen twee
keer op. Buffering zonder idempotentie is een dubbeltelling die op
zich laat wachten.

## What Changes

- **Aanmelden.** `wanderer agent-token nieuw` geeft een kortlevend
  aanmeldtoken. De agent wisselt dat bij de eerste start in voor een
  eigen geheim (`POST /agents/enrol`): de kern legt de agent vast
  (hostnaam, geheim-hash, aangemeld op) en de agent schrijft zijn
  geheim 0600 weg. Daarna is het token op.
- **Intrekken.** `wanderer agent intrekken <hostnaam>` zet de agent uit;
  vanaf dat moment worden zijn findings geweigerd, zonder dat een
  andere agent er last van heeft.
- **De kern luistert.** `serve` bouwt de router mét de
  agent-geheimenbron uit de database, zodat de route werkt zoals de
  documentatie al beschrijft. Zonder aangemelde agents blijft het
  gedrag zoals het is: weigeren.
- **Eén keer is één keer.** Elke partij findings draagt een
  identificatie; de kern accepteert die één keer en meldt een herhaling
  als "al ontvangen" in plaats van hem nog eens op te slaan.

## Scope / Not in scope

**In:** aanmelden, intrekken, het aansluiten in `serve`, en
idempotente ontvangst.

**Out (bewust):** een pakket (.deb/.rpm) of installatiescript, mTLS,
automatische updates, en centraal beheer van welke inspecteurs op een
host aanstaan. Dat is het verschil tussen een forwarder en een
vlootbeheerproduct; dit is stap één, en het staat hier zodat niemand
denkt dat het vergeten is.

## Risico dat expliciet benoemd hoort

Een aanmeldtoken is een sleutel die één keer werkt. Hij hoort kort te
leven, niet in een logregel te belanden, en niet te bestaan als er geen
agent op wacht. En "de kern luistert nu" mag geen "de kern luistert naar
iedereen" worden: zonder aangemelde agent blijft elk verzoek geweigerd.
