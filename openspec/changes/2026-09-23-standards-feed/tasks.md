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
- [ ] 2.1 netnl tenant `wanderer` on the facade; credential as a secret.
- [ ] 2.2 Import token as a secret, mounted into the Wanderer Deployment and
      the CronJob.
- [ ] 2.3 CronJob: web for the operator's own Wanderer targets, mail for the
      domain that receives mail; export findings; post to the route.

## 3. Out
- [ ] 3.1 Release, deploy, run the CronJob once by hand.
- [ ] 3.2 Nagemeten: the standards rules on `westerweel.work` score from real
      findings, not "niet gemeten"; `rijksoverheid.nl` still reads "niet
      gemeten".
- [ ] 3.3 Archive.
