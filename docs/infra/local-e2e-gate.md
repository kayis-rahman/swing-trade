# Building a local, hermetic, end-to-end quality gate

This document is the method behind `bin/e2e-gate`. It is written to be reused:
the same design works for any service that ships as a database plus an API plus
a web front end, whether or not it uses Docker, Compose or this particular
repository. The Swing Trade gate is the worked example; the reasoning is the
part worth copying.

Run it with:

```bash
./dev-stack.sh gate     # or: bin/e2e-gate
echo $?                 # 0 = pass, 1 = fail, 2 = could not start
```

---

## 1. Why a gate, and why it lives here

This repository has no hosted CI. `docs/infra/ci-cd-deployment.md` records that
the old GitHub Actions definitions are archived and inactive, and
`docs/issues/012-github-actions-ci-pipeline.md` describes the pipeline that was
never adopted. Before this gate existed, the only verification a change could
receive was whatever a developer happened to run by hand — usually
`./bin/verify-changes`, which is a fast path-aware check, not a statement that
the assembled system works.

A gate is not "more tests". Unit tests prove that functions agree with
functions. A merge gate has to answer a different question: **if this branch
lands exactly as it is, does the running product still work?** That question can
only be answered against a real database, a real HTTP surface and a real
front-end bundle — assembled the way they are assembled in production.

So the gate here runs the whole system in Docker, asserts against it over HTTP
and SQL, and throws it away afterwards. It is this project's equivalent of a CI
pipeline, executed on a machine the project already owns, using the deployment
tooling the project already ships.

## 2. The layers, and what each one proves

Layers are ordered so that each one fails for a reason that is unambiguous from
the layer alone. A build failure and a database failure must never look alike.

| # | Layer | Proves | Swing Trade implementation |
|---|-------|--------|---------------------------|
| 0 | preflight | the gate can run and cannot be fooled | ports free, Docker reachable, stale stack gone |
| 1 | unit | the code agrees with itself | `./gradlew test`, `yarn test:run` |
| 2 | artifacts | the deployable units exist and are complete | `api-plain.jar` + runtime classpath, `dist/` |
| 3 | images | those artifacts package into a runnable image | `infra/Dockerfile`, `infra/dashboard/Dockerfile` |
| 4 | boot | the system starts and *becomes ready* | compose up + readiness polling |
| 5 | checks | the running system does what it claims | 15 HTTP/SQL assertions |

Two design decisions in that table are worth defending.

**The gate packages and runs the same artifacts deployment does.** Layer 3 uses
the repository's own `infra/Dockerfile` and `infra/dashboard/Dockerfile`, fed
from the same `:api:jar` + `:api:copyRuntimeDeps` output and the same
`yarn vite build` output that `dev-stack.sh stage` ships. A gate that boots a
hand-rolled `java -jar` in a different container proves something adjacent to
what ships, not what ships.

**Fail fast between layers, collect within a layer.** Layers 0–4 abort on the
first failure: there is no value in asserting endpoints against an API that did
not compile. Layer 5 runs *every* check regardless, because a developer fixing
a red gate should see all fifteen reasons at once, not rediscover them one run
at a time.

## 3. Hermeticity: the rules

Hermetic means a run's result depends on the commit, not on the machine's mood.
Five rules, each one earned by a specific failure mode:

### 3.1 Own everything: names, network, volume, ports

The gate stack is a separate compose project (`swing-trade-e2e-gate`) with its
own network, its own volume, and host ports (18080 / 18081 / 15437) chosen not
to collide with anything the dev or stage stacks publish (5435 / 8080 / 3003 /
5436 / 8081 / 8082).

Preflight refuses to start if one of its ports is already published. That check
is not defensive padding: a gate that silently pointed at a port someone else
already owns would assert against the wrong system and report a confident pass.

### 3.2 Destroy the state, do not reset it

`down -v` on the way out, and a "remove any stale project first" step on the way
in. A gate that runs against last week's database is not testing this commit. If
the code needs a pre-existing row to pass, that is a finding to report, not a
fixture to seed.

### 3.3 Turn off everything that acts on its own

The dedicated `e2e` Spring profile (`application-e2e.properties`) disables every
scheduler, the candidate scan, data ingestion, the job orchestrator's LLM stage,
notifications, and all news providers. Nothing in a gate run may be caused by
the clock rather than by the gate.

This is the single largest contributor to a non-flaky gate, and it is why the
gate gets its own profile instead of reusing `local` or `stage`.

### 3.4 Make the impossible impossible to fake

- **Liveness vs. readiness.** Spring's default readiness group can report UP
  before the datasource is connected. The gate's profile pins
  `management.endpoint.health.group.readiness.include=readinessState,db`, so
  "ready" means "ready *and* has a database".
- **Schema agreement.** `ddl-auto=validate` against Flyway-migrated schemas
  means a migration and an `@Entity` that disagree stops the boot. The schema
  check is not decoration.
- **Assert below the cache.** `GET /api/settings` reads from an in-memory cache.
  A write→read round trip through the API alone would pass even if nothing was
  persisted. The gate therefore reads the row back out of PostgreSQL with
  `psql`. This is the difference between a business-flow check and a decoration.

### 3.5 Never touch shared state

The Docker daemon here is shared with the long-lived dev and stage stacks.
Two consequences:

- The global Docker context is never mutated. Every Docker call is scoped with
  `DOCKER_HOST` on that one command. `docker context use` would change the
  behaviour of every other shell on the machine — the same constraint
  `.codex/RULES.md` records as plan E8.
- Teardown only ever addresses its own compose project. `docker compose down`
  with the gate's project file cannot match another project's containers.

## 4. Read output, not sleeps

A fixed `sleep 15` after `docker compose up` is a bet that startup is fast
enough, and it is wrong in both directions: it wastes time when startup is fast
and it races when startup is slow. Worse, `docker compose up -d` returns when
containers are *created*, which can be seconds before PostgreSQL accepts a
connection.

The gate polls a real readiness signal — `GET /actuator/health/readiness`
returning `{"status":"UP"}` — with a deadline, and prints how long it waited.
Compose healthchecks are configured too, so `depends_on: condition:
service_healthy` and `docker compose ps` are honest for anyone who inspects the
stack by hand.

Waiting has a floor and a ceiling, and the ceiling is what makes a hung gate
diagnosable: if the API container exits during startup, the gate prints the last
40 log lines and names the cause instead of waiting out the timeout.

## 5. Noticing and preventing flakiness

A gate that cries wolf gets muted within a week, and a muted gate is worse than
no gate. The defences used here:

- **No timing assertions.** The gate asserts *what the system reports*, never
  *how long it took* to report it. Only the overall deadline exists, and it is
  generous (300s) relative to observed startup (~35s).
- **No shared mutable state between checks.** The one check that writes
  (`openai.model`) restores the value it read before returning.
- **No ordering dependence.** Checks are independent, so a failure never
  cascades into misleading downstream failures.
- **Deterministic expectations.** `flow.sentiment_defaults` asserts the *empty
  database* contract (`score=NEUTRAL`, `total=0`), which is only stable because
  the database is guaranteed empty by §3.2.
- **Report skips, never fold them into the pass.** The backend layer prints
  `tests=2351 failures=0 skipped-by-design=133`. A suite that quietly stops
  running a class looks exactly like one that passes; the gate makes the skip
  count part of its normal output.
- **Fix the flake or mark it.** If a check genuinely cannot be made
  deterministic, it is listed in §7 as a known gap with what would close it —
  never quietly dropped.

## 6. What belongs in the local pass, and what needs a hosted runner

The dividing line is *time and isolation*, not importance.

| Goes in the local gate | Needs a hosted runner |
|-----------------------|-----------------------|
| Runs in minutes on one machine | Takes hours, or needs many cores |
| Fits in one Docker host's memory | Needs its own VM or more RAM than a laptop has |
| Fully deterministic, no network | Needs real provider credentials or the public internet |
| Reproducible from a clean checkout | Should run on *every* push, including branches nobody reviews |

Concretely for a project of this shape:

**Local gate (this gate):** compile, unit and integration tests, image build,
schema migration, boot, read-only and reversible-write endpoint assertions,
front-end bundle served and proxying correctly.

**Hosted runner (a later step, not this gate):** the full Playwright browser
suite against a real deployment, long-running scans and backfills, anything
calling LLM or broker providers with real credentials, multi-architecture
image builds, and long-running soak/memory checks. Two of Swing Trade's existing
Playwright specs are `*.live.spec.ts` precisely because they need a real
deployment.

The rule of thumb: **if a check cannot pass on a laptop in under ten minutes
without credentials, it is not a local gate check.** Put it in the hosted tier
and say so, rather than making the local gate slower and flakier for everyone.

## 7. What this gate does not prove

Stated plainly, because the value of a gate is the trust you place in it.

- **No real providers.** No LLM inference, no market-data vendor, no broker
  session, no Telegram or Signal delivery. The `e2e` profile points every
  provider at a dead address so such a call fails fast instead of stalling.
  *Closing this:* a separate credentialed tier on a runner.
- **No browser.** The dashboard is verified as served HTML, a built JS bundle
  that returns 200, SPA deep-link fallback, and an nginx→API write round trip.
  It is not driven by a browser. The repository's Playwright suite covers
  rendering; wiring it to run against this stack (with a browser image in
  compose) is the obvious next layer.
- **No production packaging.** The gate runs the JAR-based stage image. The
  GraalVM native binary (`runtime-native`) is never built or exercised here;
  it is also never exercised by any test, so native-image-only defects remain
  invisible. *Closing this:* build the native image in the gate on a runner
  with enough memory for `-J-Xmx4g`.
- **Not `check`.** The gate runs `./gradlew test`, not `./gradlew check`. `check`
  additionally runs checkstyle, PMD and the JaCoCo 80% coverage verification
  that `backend/build.gradle.kts` deliberately wires in and that currently fails
  (measured coverage is far below the threshold in every module — the repository
  comment says this is intentional so the gap is visible rather than silent).
  Running it here would mean a gate that is always red for a known, tracked
  reason. *Closing this:* raise real coverage, then switch the gate to `check`.
- **Two Gradle test source sets are outside the gate.** `integrationTest`
  (wired into `check`) is not run. Nor are the 133 `@Disabled` tests in the
  `test` suite — among them `PositionControllerTest`, disabled with the note
  that it needs PostgreSQL on `localhost:5432`. The gate reports the skip count
  so the gap stays visible.
- **The gate runs on the shared pi-node Docker host.** That host also serves the
  stage deployment, Home Assistant, Pi-hole and others. The gate is isolated by
  project name, network and ports, but it is not a dedicated machine; a host
  under heavy load elsewhere can slow it down. Its own memory limit (1200 MB) is
  set to keep it from starving the stage stack.
- **The dashboard Vitest suite is timeout-sensitive under host load.** Observed
  directly: with the build machine's load average above ~200 (other tenants'
  work, not this gate), 5 of 59 Vitest files blew through the default 5000 ms
  per-test timeout and the suite failed; the identical suite passed standalone
  minutes later. This is reported, not fixed — raising `testTimeout` to make the
  gate green would mask genuine hangs, and nothing here should weaken an existing
  test to pass. *Closing this:* set an explicit `testTimeout` in
  `dashboard/vitest.config.ts` sized for a loaded machine and a dedicated runner,
  and split the heaviest jsdom suites. Until then, treat a dashboard-suite
  timeout failure as "re-run on a quiet machine" rather than as a code defect —
  and treat a *repeated* one as a real defect.

## 8. Porting this to another project

The recipe, in order:

1. **Define the pass and the exit code.** One command, 0 or 1, and a line of
   output naming what failed. Decide up front whether it fails fast between
   layers or runs everything.
2. **Name the layers** — the table in §2 is the template. For each, write down
   what a pass proves. If you cannot state it in one sentence, the layer is
   testing nothing in particular.
3. **Build the stack from the deployable artifacts**, using the same images
   deployment uses. A separate "test-only" image proves the wrong thing.
4. **Give it its own identity**: compose project, network, volume, ports, and a
   config file no other stack shares.
5. **Write a hermetic profile/config** that disables every background job, every
   outbound call, and every credential-gated feature.
6. **Replace sleeps with readiness probes**, with a deadline, and a defined
   behaviour when the service exits during startup.
7. **Assert below the cache** on every business flow: after the write, read the
   state from the system of record, not from the process that wrote it.
8. **Prove the boundary is enforced**: an unauthenticated request must fail, not
   merely succeed with the right one.
9. **Tear down on `EXIT`, always**, volumes included, and capture logs to a file
   *before* tearing down so a failure is still diagnosable.
10. **List what it cannot prove**, in the documentation, with what would close
    each gap. Then keep that list true.

