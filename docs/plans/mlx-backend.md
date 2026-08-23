# Plan: Add MLX Backend to SwingTrade LLM Pipeline

## What's Already Done

- Three backend modes exist: `local` (llama.cpp on Mac), `pi_ssh` (llama.cpp on Pi), `openai` (GPUHub/cloud)
- `SpringAiLlmClient` wraps Spring AI's `ChatClient` — works with any OpenAI-compatible endpoint
- `LlmConfig` creates three `OpenAiChatModel` beans, all reading config from `AppSettingsStore` (DB)
- `LlmClientProvider` routes to the selected backend at runtime
- `LlmServerManager` interface + two implementations (`LlamaCppServerManager`, `PiLlamaServerManager`) manage server lifecycle
- `SettingsController` exposes REST endpoints for backend selection, server lifecycle, and inference testing
- `SettingsView.vue` has three backend tabs with config panels
- `settings.ts` store holds `llmBackend: 'local' | 'pi_ssh' | 'gpuhub'`

## Refactoring Done (gpuhub → OpenAI)

- `LlmBackendSelector.java`: Removed `gpuhub` alias in `fromKey()`
- `LlmConfig.java`: Removed `gpuhub.api_key` fallback in `resolveApiKey()`
- `SettingsController.java`: Renamed `/settings/gpuhub` → `/settings/openai`, `gpuhub.api_key` → `openai.api_key`, updated `saveAllSettings`
- `settings.ts` (store): Renamed `gpuhub` → `openai` in union types
- `settings.ts` (API): Renamed `getGpuHubSettings` → `getOpenAiSettings`, `setGpuHubSettings` → `setOpenAiSettings`
- `index.ts` (API): Updated exports
- `SettingsView.vue`: Renamed `gpuhub` → `openai` in backend tabs and config panel
- Backend compiles successfully

## What Needs to Be Done

### Step 1: Add MLX enum to backend selector

**File:** `backend/llm/src/main/java/com/swingtrade/llm/service/LlmBackendSelector.java`

Add `MLX("mlx")` to the `Backend` enum after `PI_SSH`. No other changes needed — `fromKey()` already handles all enum values.

### Step 2: Create MlxServerManager

**New file:** `backend/llm/src/main/java/com/swingtrade/llm/service/MlxServerManager.java`

Implements `LlmServerManager`. Lightweight process tracker:

- **Start:** Launches `mlx_lm.server --model <model> --port 8081` via `ProcessBuilder`
- **Health check:** Polls `http://<wlan-ip>:8081/health` (mlx_lm.server exposes this)
- **WLAN IP:** Configurable via `mlx.server.url` property (e.g., `http://192.168.1.50:8081`). Detected automatically or set manually.
- **PID tracking:** Stores PID in `~/.swingtrade/mlx.pid`
- **Stop:** Kills process by PID or port
- **IsRunning:** Checks PID file + port socket + health endpoint

Key differences from `LlamaCppServerManager`:
- No model path on Pi — model name is a simple string (e.g., `Qwen/Qwen2.5-3B-Instruct`)
- No SSH — local process only
- Uses `mlx_lm.server` instead of `llama-server`
- Health endpoint at `/health` vs llama.cpp's `/health`

### Step 3: Add MLX chat model bean

**File:** `backend/llm/src/main/java/com/swingtrade/llm/config/LlmConfig.java`

Add `mlxChatModel` bean:

```java
@Bean
public OpenAiChatModel mlxChatModel(
        AppSettingsStore appSettingsStore,
        @Value("${spring.ai.openai.api-key:none}") String apiKey) {
    return createChatModel(appSettingsStore, "mlx.server.url",
            "http://192.168.1.50:8081", "mlx.model", "Qwen/Qwen2.5-3B-Instruct", apiKey);
}
```

Properties to add:
- `mlx.server.url` — MLX server URL (default: `http://192.168.1.50:8081`; set to your WLAN IP)
- `mlx.model` — model name (default: `Qwen/Qwen2.5-3B-Instruct`; MLX-compatible Qwen2.5 3B Q4, ~2GB RAM)

### Step 4: Wire MLX into LlmClientProvider

**File:** `backend/llm/src/main/java/com/swingtrade/llm/service/LlmClientProvider.java`

Add `@Qualifier("mlxChatModel") OpenAiChatModel mlxModel` constructor param. Add `case MLX -> mlxModel` to the switch.

### Step 5: Wire MLX into SettingsController

**File:** `backend/api/src/main/java/com/swingtrade/api/controller/SettingsController.java`

Changes:
1. Inject `MlxServerManager` instead of `LlamaCppServerManager`
2. Update `setLlmSettings` restart logic: `case MLX -> mlxServerManager`
3. Add `/settings/mlx/start`, `/settings/mlx/stop`, `/settings/mlx/status` endpoints (same pattern as Pi endpoints)
4. Add `/settings/test/mlx` endpoint for inference testing
5. Update `getLlmSettings` to return `mlx.model`

### Step 6: Update frontend settings store

**File:** `dashboard/src/stores/settings.ts`

Changes:
1. `LlmSettings` interface: add `mlxBackend: 'local' | 'pi_ssh' | 'openai' | 'mlx'` (gpuhub renamed to openai)
2. Add `mlxModel: string` field (default: `Qwen/Qwen2.5-3B-Instruct`; MLX-compatible Qwen2.5 3B Q4, ~2GB RAM)
3. Update `loadAll()` to load `mlx.model` from settings
4. Update `saveSettings()` and `saveLlmSettings()` to include `mlx.model`

### Step 7: Add MLX API client functions

**File:** `dashboard/src/api/settings.ts`

Add:
- `startMlxServer()` — POST `/settings/mlx/start`
- `stopMlxServer()` — POST `/settings/mlx/stop`
- `getMlxServerStatus()` — GET `/settings/mlx/status`
- `testMlxConnection()` — POST `/settings/test/mlx`

Export all from `dashboard/src/api/index.ts`.

### Step 8: Add MLX tab to SettingsView

**File:** `dashboard/src/views/SettingsView.vue`

Changes:
1. Add `mlx` to `llmBackends` array: `{ value: 'mlx', label: 'MLX' }`
2. Add MLX config panel (same structure as Pi panel):
   - Model name input (dropdown with common MLX models)
   - Start/Stop/Refresh buttons
   - Server status indicator
   - Test inference button
3. Add state refs: `mlxServerRunning`, `mlxLoading`, `mlxTestResult`, `mlxTestSuccess`
4. Add handler functions: `handleMlxStart`, `handleMlxStop`, `refreshMlxStatus`, `testMlxConnection`
5. Update `onMounted` to call `refreshMlxStatus`

### Step 9: Update application.properties

**File:** `backend/llm/src/main/resources/application.properties`

Add:
```properties
mlx.model=Qwen/Qwen2.5-3B-Instruct
mlx.server.url=http://192.168.1.50:8081
```

**File:** `backend/api/src/main/resources/application-local.properties`

No changes needed — defaults are fine.

## Files Changed

| File | Type | Change |
|------|------|--------|
| `LlmBackendSelector.java` | Modify | Refactor gpuhub→openai, add `MLX` enum |
| `MlxServerManager.java` | **New** | MLX server lifecycle manager |
| `LlmConfig.java` | Modify | Add `mlxChatModel` bean |
| `LlmClientProvider.java` | Modify | Wire `mlxModel` |
| `SettingsController.java` | Modify | Refactor gpuhub→openai, add MLX endpoints |
| `settings.ts` (store) | Modify | Refactor gpuhub→openai, add `mlx` backend, `mlx.model` |
| `settings.ts` (API) | Modify | Refactor gpuhub→openai, add MLX lifecycle functions |
| `index.ts` (API) | Modify | Export MLX functions, rename gpuhub→openai |
| `SettingsView.vue` | Modify | Refactor gpuhub→openai, add MLX tab panel |
| `application.properties` (llm) | Modify | Add `mlx.model`, `mlx.server.url` |

## Verification

1. `./gradlew :llm:compileJava :api:compileJava` — backend compiles ✅
2. `cd dashboard && yarn build` — frontend compiles (pre-existing errors only)
3. Start `mlx_lm.server --model Qwen/Qwen2.5-3B-Instruct --port 8081` manually
4. Select "MLX" in Settings → AI/LLM tab
5. Click "Test" — should return success

## Implementation Status

All steps implemented and verified:
- Step 0: gpuhub → OpenAI refactoring ✅ (completed in prior round)
- Step 1: MLX enum added ✅
- Step 2: MlxServerManager created ✅
- Step 3: mlxChatModel bean added ✅
- Step 4: MLX wired into LlmClientProvider ✅
- Step 5: MLX wired into SettingsController ✅ (fixed duplicate openai.api_key, added mlx.server.url)
- Step 6: Frontend settings store updated ✅
- Step 7: MLX API client functions added ✅
- Step 8: MLX tab in SettingsView ✅
- Step 9: application.properties updated ✅