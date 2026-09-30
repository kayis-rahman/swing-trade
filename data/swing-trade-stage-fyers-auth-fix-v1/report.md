# Swing Trade Stage — Fyers Auth Error Fix

**Date:** 2026-09-30
**Branch:** fm/swing-trade-stage-fyers-auth-fix-v1
**Predecessor:** data/swing-trade-stage-health-review-v1/report.md (commit d0c62fd6, deployed)

---

## 1. Reproduction (read-only, against real staging)

**Target:** staging dashboard `http://piworm.local:8082` (pi-node), API `http://piworm.local:8081`.

**Steps:**
1. Open `http://piworm.local:8082/settings` → Broker tab (default active broker: Fyers).
2. Observe broker status card: red dot, "Disconnected", and a "Connect Fyers Account" button.
3. Click "Connect Fyers Account".

**Expected:** either the Fyers OAuth login popup opens, or — since stage has no Fyers
credentials — a clear, honest state such as "Fyers is not configured on this environment".

**Observed:** a red banner appears at the top of the settings screen:
**"Fyers authentication failed. Please try again. ×"** (SettingsView.vue line ~150,
`authResultBanner === 'error'`). Dismissing the banner and clicking the button again
reproduces it deterministically — **repeatable**.

**Network evidence (browser network log):** each click issues
`GET /api/fyers/login` → **HTTP 400** `{"error":"FYERS_CLIENT_ID not configured"}`.
`GET /api/fyers/status` returns 200 `{"clientId":"","connected":false}`.

**Screenshot:** captured during reproduction (banner visible).

No staging host state was modified; only HTTP reads and browser interaction.

---

## 2. Trigger vs. masking condition vs. symptom

| Layer | Finding |
|-------|---------|
| **Initiating trigger** | User clicks "Connect Fyers Account" → `startFyersAuth()` → `GET /api/fyers/login`. |
| **Backend behavior (correct)** | `FyersAuthController.getLoginUrl()` returns 400 `{"error":"FYERS_CLIENT_ID not configured"}` because `FyersConfig.clientId` is blank on stage. Stage has **no Fyers credentials** (`env/.env.stage` has none; confirmed by the 400 body and by the health review). |
| **Masking condition (the bug)** | `startFyersAuth()`'s catch block maps **any** error to `authResultBanner = 'error'`, and the banner template renders every non-success value as "Fyers authentication failed. Please try again." The UI cannot distinguish "not configured" (400, `FYERS_CLIENT_ID not configured`) from a genuine authentication failure. The same masking exists in `submitAuthCode()` (toast, line ~1145). |
| **Visible symptom** | Red "Fyers authentication failed. Please try again. ×" banner. |

**Known-working comparison path:** on the same staging env, `GET /api/fyers/status`
returns 200 and the UI correctly renders a neutral "Disconnected" state — proving the UI
*can* show a non-error Fyers state; only the login-flow error path misreports. On a
credentials-configured environment (dev), the same click returns a login URL and opens
the OAuth popup; the 400 "not configured" response is stage's correct backend outcome.

**Conclusion:** this is **not** a genuine auth or token-refresh defect. The message is the
correct outcome (authentication cannot succeed) for the wrong reason (missing
configuration is reported as a failure). The earlier health review fixed the log noise
but not this user-visible misreport.

---

## 3. Counterfactual and falsification evidence

**Leading explanation:** the banner text is produced by an indiscriminate catch block;
the backend already tells the UI the real reason ("not configured", HTTP 400).

**Counterfactual:** if the UI distinguished the 400 "not configured" response from real
auth failures, the staging user would see "not configured" instead of "authentication
failed". The code path confirms this is the only branch taken on staging: the 400 body
is exactly `{"error":"FYERS_CLIENT_ID not configured"}`, and `apiRequest` surfaces it as
an `AppError` with `status: 400`, `kind: 'validation'`, `message: "FYERS_CLIENT_ID not configured"`.

**Falsification checks (what would disprove the explanation):**
- A genuine auth defect would show a *different* backend response (e.g. 401/500, or a
  token-exchange failure on `/api/fyers/auth`). Observed: only the 400 not-configured
  response; `/api/fyers/status` is healthy (200).
- A UI bug independent of the backend would reproduce on dev with credentials. The
  dev flow returns a login URL and opens the popup — no banner.
- If the banner were caused by the status check failing, `fyersStatusError` would be
  true and the "Connect" button would not render at all. Observed: button renders,
  status shows "Disconnected" normally.

None of these falsify the leading explanation; all evidence supports it.

---

## 4. Fix

**Smallest correct fix:** make the "not configured" state visible instead of masking it
as an authentication failure.

1. **Backend** (`FyersAuthController.java`): add a machine-readable
   `"code": "FYERS_NOT_CONFIGURED"` to the two 400 responses so the frontend can detect
   the state without matching on message text.
2. **Frontend** (`SettingsView.vue`): detect the not-configured error in
   `startFyersAuth()` and `submitAuthCode()` and render a distinct warning-state
   message — "Fyers is not configured on this environment. Contact an administrator to
   enable broker access." — instead of the red "authentication failed" banner/toast.
   Genuine failures keep the existing error message.
3. **Regression test:** mounts SettingsView, mocks `GET /api/fyers/login` → 400 with
   code `FYERS_NOT_CONFIGURED`, clicks "Connect Fyers Account", and asserts the
   not-configured message appears and "Fyers authentication failed" does not. A second
   test asserts a genuine (non-not-configured) failure still shows the original error.

**Not done (out of scope):** configuring Fyers credentials on stage (requires secrets
the worker must not handle), changing the status endpoint, or touching the disabled
ingestion jobs.

---

## 5. Verification

- [x] Reproduced on live staging dashboard (browser), repeatable, with network evidence.
- [x] Root cause isolated to frontend error mapping; backend 400 is correct.
- [x] Regression tests pass: 3 new frontend tests (not-configured via code, not-configured
  via message-only fallback, genuine failure keeps original message) — full frontend
  suite 359/359 green, typecheck + lint clean.
- [x] Backend test added (3 tests: missing clientId, missing secretKey, configured
  happy path) — all pass; `:api:compileJava` clean.
- [x] End-to-end verification against the **real staging API** (no deploy): ran the
  fixed dashboard locally with the Vite proxy pointed at `piworm.local:8081`, clicked
  "Connect Fyers Account" — the UI now shows "Fyers is not configured on this
  environment. Contact an administrator to enable broker access." (warning styling)
  instead of "Fyers authentication failed. Please try again.". This also proves the
  message-only fallback works against the current staging backend, which does not yet
  send the `code` field.
- [ ] no-mistakes pipeline green, PR opened.

**Live vs. test summary:** verified live — the original error on staging (browser,
repeatable, network log) and the fixed message end-to-end against the real staging
API. Verified by tests — the backend 400 contract and the frontend state mapping.
Not verified live — a real Fyers OAuth login (no credentials on stage; per instructions,
no credentials were invented or handled).
