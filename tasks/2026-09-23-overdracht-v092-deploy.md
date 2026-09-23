# Overdracht — v0.9.2 uitgebracht, deploy nog te doen

Getagd en gepusht: **v0.9.2** (copyrighthouder in LICENSE is nu
Mark Westerweel). Gates vóór de tag: go build/vet/test groen,
golangci-lint nul, `make playwright` 45/45.

De keten loopt vanzelf: tag -> goreleaser -> release-dispatch ->
image `core-v0.9.2` in ghcr. Wat een mens/agent nog moet doen zodra
dat image klaar is:

```sh
# 1. wachten tot beide groen zijn
gh run list -R MWest2020/Wanderer -w release -L1
gh api repos/MWest2020/wanderer-exapp/actions/runs --jq '.workflow_runs[0].conclusion'

# 2. digest ophalen
TOKEN=$(curl -s "https://ghcr.io/token?scope=repository:mwest2020/wanderer-exapp:pull&service=ghcr.io" | jq -r .token)
curl -sI -H "Authorization: Bearer $TOKEN" \
  -H "Accept: application/vnd.docker.distribution.manifest.v2+json" \
  https://ghcr.io/v2/mwest2020/wanderer-exapp/manifests/core-v0.9.2 \
  | grep -i '^docker-content-digest'

# 3. homelab bumpen (LET OP: controleer eerst `git branch --show-current`,
#    een andere sessie deelt deze working tree)
#    cluster-config/infra/wanderer/deployment.yaml regel 78
#    huidige digest: sha256:a0d37be236078d68b57bf4823d85218276b3ee6f594f6b0bc8eedff71086242e (v0.9.1)

# 4. ArgoCD laten synchroniseren en nameten
ssh jumpy "kubectl -n argocd patch application wanderer --type merge \
  -p '{\"operation\":{\"sync\":{\"revision\":\"HEAD\"},\"initiatedBy\":{\"username\":\"agent\"}}}'"
ssh jumpy "kubectl -n wanderer rollout status deploy/wanderer --timeout=180s"
ssh jumpy "kubectl -n wanderer exec deploy/wanderer -- /usr/local/bin/wanderer version"
curl -s -o /dev/null -w '%{http_code}\n' https://wanderer.westerweel.work/demo
```

## Twee dingen om te onthouden
- `npx playwright test` bouwt en seedt NIET. Altijd `make playwright`,
  anders draai je tegen een oude `bin/wanderer`.
- `~/Wanderer` en `~/homelab` worden gedeeld met een andere sessie.
  Controleer `git branch --show-current` vóór elke commit en
  `git log --oneline origin/main -1` ná elke push.
