# Proposal: CAA echt opzoeken, en de boom omhoog lopen

## Why

`netResolver.LookupCAA` geeft altijd `nil, nil` terug:

```go
func (n *netResolver) LookupCAA(_ context.Context, _ string) ([]CAA, error) {
	return nil, nil
}
```

De probe maakt daarvan `no CAA records`, en
`wand.operationeel.caa_restricts_issuance` oordeelt afhankelijk. Voor
**elk** domein, altijd. Nagemeten op 2026-09-21: digid.nl,
mijnoverheid.nl en ncsc.nl kregen alle drie "no CAA records — any public
CA may issue", terwijl ze alle drie CAA-records hebben (certsign.ro en
digicert.com; ncsc.nl heeft er ook een iodef bij).

Dit is de ergste soort fout die een scanner kan hebben: een stellige
uitspraak die nergens waar is, en die bij iedereen hetzelfde luidt. Wie
hem gelooft, gaat iets repareren dat al goed stond.

## En een tweede fout die eronder zit

CAA is hiërarchisch (RFC 8659 §3): vindt een CA niets op
`iam.westerweel.work`, dan kijkt hij bij `westerweel.work`, en zo verder
omhoog. Een implementatie die alleen de gevraagde naam bevraagt, meldt
"geen CAA" voor élk subdomein van een zone die het keurig geregeld
heeft. Onze eigen subdomeinen zijn daar het voorbeeld van: de records
staan op de apex, de subdomeinen erven ze.

## What Changes

- `LookupCAA` doet een echte DNS-vraag naar CAA-records.
- De opzoeking loopt de boom omhoog tot ze records vindt of bij het
  registreerbare domein is, en legt vast op wélke naam ze gevonden zijn.
- De finding zegt waar ze vandaan komen ("geërfd van westerweel.work"),
  zodat een lezer niet denkt dat het subdomein ze zelf heeft.
- Een test faalt als de resolver stilzwijgend niets teruggeeft — de
  huidige situatie zou daarmee zijn gevangen.

## Scope / Not in scope

**In:** de opzoeking, de boomwandeling, de herkomst in de finding, en
de tests.

**Out:** de regel zelf (die blijft oordelen zoals hij doet, nu op
gegevens die kloppen), en andere ontbrekende opzoekingen — als die er
zijn, is dat een eigen change.

## Wat dit ook zegt over de rest

Een functie die `nil, nil` teruggeeft is geen halve implementatie maar
een stille leugen: alles eromheen werkt, de tests slagen, en de uitvoer
is onzin. Bij het afronden hoort daarom een controle of er meer zulke
plekken zijn — niet als losse opruimactie, maar als vraag die
beantwoord moet worden voordat we deze change afsluiten.
