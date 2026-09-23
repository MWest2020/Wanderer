# Tasks: standards-feed

## 1. The route — habitat run 01
- [x] 1.1 `POST /imports/internetnl`, reusing the CLI's import path (no second
      implementation). Counts in the response.
- [x] 1.2 Token: `WANDERER_IMPORT_TOKEN`, `Authorization: Bearer`, constant-time
      compare. Unset → refuse everything, say so once at startup.
- [x] 1.3 Tests for every scenario in the spec delta, each checked once with
      its fix removed.
- [x] 1.4 docs: `docs/how-to/internetnl-ci.md` gains the HTTP route next to
      the CLI; CHANGELOG.

## 2. The feed — homelab
- [x] 2.1 netnl tenant `wanderer` on the facade; credential as a secret.
- [x] 2.2 Import token as a secret, mounted into the Wanderer Deployment and
      the CronJob.
- [x] 2.3 CronJob: web for the operator's own Wanderer targets, mail for the
      domain that receives mail; export findings; post to the route.

## 3. Out
- [x] 3.1 Release, deploy, run the CronJob once by hand. (v0.10.0; Job
      `wanderer-standards-eerste`: web 4 imported, mail 1, 0 unknown.)
- [ ] 3.2 Nagemeten: the standards rules on `westerweel.work` score from real
      findings, not "niet gemeten"; `rijksoverheid.nl` still reads "niet
      gemeten".
- [ ] 3.3 Archive — after section 4.

## 4. Import scans are not "the latest scan" — habitat run 02
Found by the first live run (3.1): the import landed after the perimeter scans
and became the newest scan of four domains, so the fleet screen showed 0/0 for
them. `PreviousScanForTarget` has the same blind spot, so drift would diff a
perimeter scan against an import. The CronJob is suspended in homelab until
this ships; the four domains were rescanned by hand.
- [x] 4.1 One store-level notion of "import scan", used by every latest-scan
      selection (fleet, dashboard, aggregate/trends, demo, door) and by
      `PreviousScanForTarget`.
- [x] 4.2 Tests for every scenario in `specs/web-ui` and `specs/scheduling` of
      this change, each checked once with its fix removed.
- [ ] 4.3 Release; unsuspend the CronJob in homelab; after its next run the
      fleet row of `westerweel.work` still shows its perimeter score.
