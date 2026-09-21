# Proposal: een demopagina die één domein laat zien, en verder niets

## Why

Mark, 2026-09-21: *"een demo page voor wanderer is geen slecht idee, net
als wordsworth. mag westerweel.work als target."*

Wie Wanderer niet kent, komt nu op een inlogscherm uit. Er is geen
manier om te laten zien wat het doet zonder iemand een account te geven.
Internet.nl lost dat op door een resultaatpagina te tonen die je kunt
delen; netnl heeft een anonieme, begrensde demo.

## De valkuil die dit ontwerp bepaalt

"De UI zonder login" kan niet. De pagina's die het meest laten zien —
de vloot, de regelpagina — tonen **andere domeinen**: welke targets een
organisatie volgt, en wie waarop faalt. Dat is precies de informatie
die je niet aan het open internet geeft, en het is ook niet van de
bezoeker.

Daarom: de demo is een **eigen, smalle route** die precies één
vooraf ingesteld domein toont, met het antwoord en de onderbouwing van
dat ene domein, en verder niets. Geen vloot, geen regelcatalogus met
andere targets, geen scanknop, geen organisaties. Het is geen uitzondering
op de inlogregel maar een apart, klein oppervlak.

## What Changes

- Eén publieke route (`/demo`) die de laatste scan van het ingestelde
  demodomein toont: het antwoord in één zin, de zeven stromen plus
  verantwoording als vragen met antwoorden, en het bewijs uitklapbaar —
  hetzelfde als een ingelogde gebruiker ziet, maar alleen voor dát
  domein.
- Het demodomein staat in de configuratie (`demo.target`), leeg =
  geen demo. Voor deze instantie: `westerweel.work`, Marks eigen domein,
  dus geen vraagstuk over toestemming.
- Links die naar de rest van de UI zouden leiden (regelpagina's, vloot,
  andere scans) zijn op de demopagina afwezig, niet alleen verborgen.
- De pagina zegt wat hij is: een voorbeeld, met de datum van de scan,
  en waar je het zelf kunt draaien.
- De publieke tunnel laat `/demo` door; al het andere blijft zoals het
  is (`/ui` achter de login, de rest dicht).

## Niet scannen op verzoek

De demo toont een scan die op schema is gemaakt, niet een die de
bezoeker start. Een knop "scan mijn domein" op een open pagina maakt van
deze instantie een scanner-voor-derden: dat is een abuse-oppervlak en
het is niet wat deze demo moet bewijzen. Wie zijn eigen domein wil
scannen, installeert Wanderer of vraagt een account.

## Scope / Not in scope

**In:** de demoroute, de instelling, de weergave, en de tunnelregel.

**Out:** scannen vanaf de demo, meerdere demodomeinen, en een publieke
regelpagina. Ook out: de demo als tweede UI onderhouden — hij hergebruikt
de bestaande weergave van antwoord en onderbouwing.
