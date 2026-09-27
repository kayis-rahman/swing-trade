<template>
  <div class="view-shell settings-shell p-4 sm:p-6 animate-fade-in">
    <div
      class="settings-header mb-6 flex flex-col gap-5 rounded-2xl border border-border-subtle p-5 sm:flex-row sm:items-end sm:justify-between sm:p-7"
    >
      <div>
        <p class="settings-kicker">
          Control centre <span aria-hidden="true">/</span> Workspace preferences
        </p>
        <h1 class="mt-2 font-display text-3xl font-semibold tracking-tight text-text-primary">
          Settings
        </h1>
        <p class="mt-2 max-w-xl text-sm leading-6 text-text-muted">
          Configure how Swing Trade connects, thinks, trades, and reports back to you.
        </p>
      </div>
      <button
        :disabled="saving || unconfirmedDefaults"
        class="settings-save rounded-xl bg-brand px-4 py-2.5 text-sm font-semibold text-brand-text transition-all hover:-translate-y-0.5 hover:bg-brand-hover disabled:opacity-50"
        :class="saved ? 'bg-success' : ''"
        @click="handleSave"
      >
        {{ saving ? 'Saving...' : saved ? 'Saved!' : 'Save All Settings' }}
      </button>
    </div>

    <div class="settings-overview mb-6 grid gap-3 sm:grid-cols-3">
      <div class="settings-status-card rounded-xl border border-border-subtle p-4">
        <span class="settings-card-label">Active broker</span>
        <span class="settings-card-value">{{
          brokers.find((b) => b.value === settings.selectedBroker)?.label
        }}</span>
      </div>
      <div class="settings-status-card rounded-xl border border-border-subtle p-4">
        <span class="settings-card-label">Intelligence</span>
        <span class="settings-card-value">{{
          llmBackends.find((b) => b.value === llmSettings.llmBackend)?.label
        }}</span>
      </div>
      <div class="settings-status-card rounded-xl border border-border-subtle p-4">
        <span class="settings-card-label">Trading mode</span>
        <span class="settings-card-value">{{
          settings.tradingConfig.mode === 'paper' ? 'Paper Trading' : 'Live Trading'
        }}</span>
      </div>
    </div>

    <!-- Section navigation -->
    <div
      role="tablist"
      aria-label="Settings sections"
      class="settings-nav mb-6 grid grid-cols-2 gap-2 rounded-2xl border border-border-subtle p-2 sm:grid-cols-5"
    >
      <button
        role="tab"
        aria-label="Broker"
        :aria-selected="activeTab === 'broker'"
        class="settings-tab px-3 py-3 text-left text-sm font-medium transition-all"
        :class="
          activeTab === 'broker'
            ? 'border-b-2 border-brand text-brand'
            : 'text-text-muted hover:text-text-primary'
        "
        @click="activeTab = 'broker'"
      >
        <span class="settings-tab-index">01</span><span>Broker</span
        ><span class="settings-tab-detail">Connection</span>
      </button>
      <button
        role="tab"
        aria-label="AI/LLM"
        :aria-selected="activeTab === 'llm'"
        class="settings-tab px-3 py-3 text-left text-sm font-medium transition-all"
        :class="
          activeTab === 'llm'
            ? 'border-b-2 border-brand text-brand'
            : 'text-text-muted hover:text-text-primary'
        "
        @click="activeTab = 'llm'"
      >
        <span class="settings-tab-index">02</span><span>AI / LLM</span
        ><span class="settings-tab-detail">Intelligence</span>
      </button>
      <button
        role="tab"
        aria-label="Trading"
        :aria-selected="activeTab === 'trading'"
        class="settings-tab px-3 py-3 text-left text-sm font-medium transition-all"
        :class="
          activeTab === 'trading'
            ? 'border-b-2 border-brand text-brand'
            : 'text-text-muted hover:text-text-primary'
        "
        @click="activeTab = 'trading'"
      >
        <span class="settings-tab-index">03</span><span>Trading</span
        ><span class="settings-tab-detail">Risk & limits</span>
      </button>
      <button
        role="tab"
        aria-label="Health"
        :aria-selected="activeTab === 'health'"
        class="settings-tab px-3 py-3 text-left text-sm font-medium transition-all"
        :class="
          activeTab === 'health'
            ? 'border-b-2 border-brand text-brand'
            : 'text-text-muted hover:text-text-primary'
        "
        @click="activeTab = 'health'"
      >
        <span class="settings-tab-index">04</span><span>Health</span
        ><span class="settings-tab-detail">Diagnostics</span>
      </button>
      <button
        role="tab"
        aria-label="Scanning"
        :aria-selected="activeTab === 'scanning'"
        class="settings-tab px-3 py-3 text-left text-sm font-medium transition-all"
        :class="
          activeTab === 'scanning'
            ? 'border-b-2 border-brand text-brand'
            : 'text-text-muted hover:text-text-primary'
        "
        @click="activeTab = 'scanning'"
      >
        <span class="settings-tab-index">05</span><span>Scanning</span
        ><span class="settings-tab-detail">Candidate Explorer</span>
      </button>
    </div>

    <div class="settings-content w-full max-w-none">
      <div
        v-if="unconfirmedDefaults"
        role="status"
        class="mb-4 rounded-lg bg-warning-bg p-3 text-sm text-warning"
      >
        Some settings could not be loaded; showing unconfirmed defaults. Retry before saving.
      </div>
      <!-- Auth Success/Error Banner -->
      <div
        v-if="authResultBanner"
        class="rounded-lg p-4 text-sm font-medium"
        :class="
          authResultBanner === 'success' ? 'bg-success-bg text-success' : 'bg-danger-bg text-danger'
        "
      >
        {{
          authResultBanner === 'success'
            ? 'Fyers connected successfully!'
            : 'Fyers authentication failed. Please try again.'
        }}
        <button class="ml-2 opacity-60 hover:opacity-100" @click="authResultBanner = null">
          &times;
        </button>
      </div>

      <!-- Broker Tab -->
      <div v-show="activeTab === 'broker'">
        <div class="broker-card card-panel p-5 sm:p-6">
          <div class="broker-card-heading">
            <div>
              <p class="settings-section-kicker">Execution access</p>
              <h2 class="mt-2 text-xl font-semibold tracking-tight text-text-primary">
                Broker Connection
              </h2>
              <p class="mt-1 max-w-lg text-sm leading-6 text-text-muted">
                Choose the account used for market data and order execution.
              </p>
            </div>
            <span class="broker-card-mark">01</span>
          </div>

          <!-- Broker Selection -->
          <div class="broker-options mt-6">
            <button
              v-for="b in brokers"
              :key="b.value"
              class="broker-option rounded-xl border p-4 text-left text-sm font-medium transition-all"
              :class="
                settings.selectedBroker === b.value
                  ? 'border-brand bg-brand-subtle text-brand'
                  : 'border-border-subtle text-text-muted hover:border-border-default hover:text-text-primary'
              "
              @click="settings.selectedBroker = b.value"
            >
              <span class="flex items-center justify-between gap-3">
                <span>{{ b.label }}</span>
                <span
                  v-if="settings.selectedBroker === b.value"
                  class="broker-option-check"
                  aria-hidden="true"
                >
                  ✓
                </span>
              </span>
              <span class="mt-1 block text-xs font-normal text-text-muted">
                {{ b.value === 'fyers' ? 'Market data and trading' : 'Integration coming soon' }}
              </span>
            </button>
          </div>

          <!-- Connection Status -->
          <div
            v-if="settings.selectedBroker === 'fyers'"
            class="broker-status mt-4 rounded-xl border p-4"
            :class="
              fyersConnected
                ? 'border-success/50 bg-success-bg'
                : 'border-border-subtle bg-bg-primary/50'
            "
          >
            <div class="flex items-center justify-between gap-4">
              <div class="flex min-w-0 items-center gap-3">
                <span v-if="fyersStatusError" class="text-sm font-medium text-warning"
                  >Status unavailable</span
                >
                <span
                  v-else
                  class="h-2.5 w-2.5 rounded-full"
                  :class="fyersConnected ? 'bg-success pulse-dot' : 'bg-danger'"
                />
                <span
                  v-if="!fyersStatusError"
                  class="text-sm font-semibold"
                  :class="fyersConnected ? 'text-success' : 'text-text-muted'"
                >
                  {{ fyersConnected ? 'Connected' : 'Disconnected' }}
                </span>
              </div>
              <span
                v-if="fyersConnected && fyersStatus?.clientId"
                class="truncate font-mono text-[11px] text-text-muted"
              >
                {{ fyersStatus.clientId }}
              </span>
            </div>
            <p class="mt-2 text-xs text-text-muted">
              {{
                fyersConnected ? 'Ready for authenticated requests.' : 'Authentication required.'
              }}
            </p>
          </div>

          <!-- Connect Button -->
          <div
            v-if="settings.selectedBroker === 'fyers' && !fyersConnected && !fyersStatusError"
            class="broker-actions mt-4 space-y-3"
          >
            <button
              :disabled="authing"
              class="w-full rounded-md bg-brand px-4 py-2.5 text-sm font-semibold text-brand-text transition-colors hover:bg-brand-hover disabled:opacity-50"
              @click="startFyersAuth"
            >
              {{ authing ? 'Opening Fyers...' : 'Connect Fyers Account' }}
            </button>

            <!-- Manual Auth Code -->
            <div v-if="showAuthCodeInput" class="flex gap-2">
              <input
                v-model="authCodeInput"
                placeholder="Paste auth code from browser"
                class="flex-1 rounded-md border border-border-subtle bg-bg-primary px-3 py-2 text-sm text-text-primary placeholder:text-text-muted focus:border-brand focus:outline-none"
                @keydown.enter="submitAuthCode"
              />
              <button
                :disabled="authing"
                class="rounded-md border border-brand bg-brand-subtle px-4 py-2 text-sm font-medium text-brand transition-colors hover:bg-brand/20 disabled:opacity-50"
                @click="submitAuthCode"
              >
                Submit
              </button>
            </div>
            <p v-if="showAuthCodeInput" class="text-xs text-text-muted">
              Complete login on Fyers, then paste the auth code here.
            </p>
          </div>

          <!-- Disconnect -->
          <div v-if="settings.selectedBroker === 'fyers' && fyersConnected" class="mt-4">
            <button
              class="rounded-md border border-danger/30 bg-danger-bg px-4 py-2 text-sm font-medium text-danger transition-colors hover:bg-danger/10"
              @click="disconnectFyers"
            >
              Disconnect
            </button>
          </div>

          <!-- Upstox Placeholder -->
          <div
            v-if="settings.selectedBroker === 'upstox'"
            class="mt-4 rounded-lg border border-border-subtle p-4 text-center"
          >
            <p class="text-sm text-text-muted">Upstox integration coming soon.</p>
          </div>
        </div>
      </div>

      <!-- LLM Tab -->
      <div v-show="activeTab === 'llm'">
        <div class="llm-card card-panel p-5 sm:p-6">
          <div class="settings-panel-heading">
            <div>
              <p class="settings-section-kicker">Decision support</p>
              <h2 class="mt-2 text-xl font-semibold tracking-tight text-text-primary">
                LLM & Intelligence
              </h2>
              <p class="mt-1 max-w-lg text-sm leading-6 text-text-muted">
                Configure the services that enrich signals with sentiment and research.
              </p>
            </div>
            <span class="settings-panel-mark">02</span>
          </div>

          <!-- Backend Selection -->
          <div class="llm-backend-section mt-6 mb-6">
            <h3 class="mb-3 text-sm font-semibold text-text-primary">LLM Backend</h3>
            <div class="llm-backend-options">
              <button
                v-for="b in llmBackends"
                :key="b.value"
                class="llm-backend-option rounded-xl border p-4 text-left text-sm font-medium transition-all"
                :class="
                  llmSettings.llmBackend === b.value
                    ? 'border-brand bg-brand-subtle text-brand'
                    : 'border-border-subtle text-text-muted hover:border-border-default hover:text-text-primary'
                "
                @click="llmSettings.llmBackend = b.value"
              >
                <span class="flex items-center justify-between gap-3">
                  <span>{{ b.label }}</span>
                  <span
                    v-if="llmSettings.llmBackend === b.value"
                    class="broker-option-check"
                    aria-hidden="true"
                  >
                    ✓
                  </span>
                </span>
              </button>
            </div>

            <!-- Backend descriptions -->
            <div class="space-y-2 text-xs text-text-muted">
              <div v-if="llmSettings.llmBackend === 'local'" class="rounded-md bg-bg-primary p-3">
                llama.cpp server running on the same machine as this app.
              </div>
              <div
                v-else-if="llmSettings.llmBackend === 'pi_ssh'"
                class="rounded-md bg-bg-primary p-3"
              >
                llama.cpp server on Pi (dietpi@piworm). Java starts/stops it via SSH.
                <span class="text-text-secondary">Port:</span> 8089
              </div>
              <div
                v-else-if="llmSettings.llmBackend === 'openai'"
                class="rounded-md bg-bg-primary p-3"
              >
                External OpenAI-compatible LLM endpoint. No server management needed.
              </div>
              <div
                v-else-if="llmSettings.llmBackend === 'ollama'"
                class="rounded-md bg-bg-primary p-3"
              >
                Local Ollama server for sentiment analysis. Runs entirely on your machine.
              </div>
              <div
                v-else-if="llmSettings.llmBackend === 'laya'"
                class="rounded-md bg-bg-primary p-3"
              >
                Remote Laya backend (OpenAI-compatible), configured from the dashboard. No server
                management needed.
              </div>
              <div
                v-else-if="llmSettings.llmBackend === 'pi_agent'"
                class="rounded-md bg-bg-primary p-3"
              >
                Local Pi CLI agent using its own authenticated provider session. Swing Trade never
                stores the provider token.
              </div>
            </div>
          </div>

          <!-- Local LLM (llama.cpp) -->
          <div v-show="llmSettings.llmBackend === 'local'" class="space-y-4 mb-6">
            <h3 class="text-sm font-medium text-text-secondary">Local LLM (llama.cpp)</h3>

            <div class="flex gap-2">
              <input
                v-model="llmSettings.llmBaseUrl"
                placeholder="http://localhost:8080/v1"
                class="flex-1 rounded-md border border-border-subtle bg-bg-primary px-3 py-2 text-sm text-text-primary placeholder:text-text-muted focus:border-brand focus:outline-none"
              />
            </div>

            <div class="flex gap-2">
              <input
                v-model="llmSettings.llamacppModel"
                placeholder="Path to the GGUF model configured on the server"
                class="flex-1 rounded-md border border-border-subtle bg-bg-primary px-3 py-2 text-sm text-text-primary placeholder:text-text-muted focus:border-brand focus:outline-none"
              />
              <span class="self-center text-xs text-text-muted">Model</span>
            </div>

            <p class="text-xs text-text-muted">
              Used for sentiment analysis on every signal. Switching models requires restarting the
              llama.cpp service.
            </p>
          </div>

          <!-- Pi SSH LLM -->
          <div v-show="llmSettings.llmBackend === 'pi_ssh'" class="space-y-4 mb-6">
            <h3 class="text-sm font-medium text-text-secondary">Pi SSH LLM (llama.cpp)</h3>
            <p class="text-xs text-text-muted">
              llama.cpp server on Pi 5 (dietpi@piworm). Java starts/stops it via SSH on port 8089.
            </p>
            <div class="flex gap-2">
              <input
                v-model="llmSettings.llamacppModel"
                placeholder="Path to the GGUF model configured on the server"
                class="flex-1 rounded-md border border-border-subtle bg-bg-primary px-3 py-2 text-sm text-text-primary placeholder:text-text-muted focus:border-brand focus:outline-none"
              />
              <span class="self-center text-xs text-text-muted">Model</span>
            </div>
            <p class="text-xs text-text-muted">
              Model path on the Pi. Switching models requires restarting the llama.cpp service.
            </p>
            <!-- Server lifecycle -->
            <div class="flex items-center gap-3 pt-2">
              <div class="flex items-center gap-2">
                <span
                  class="h-2.5 w-2.5 rounded-full"
                  :class="piServerRunning ? 'bg-success' : 'bg-danger'"
                />
                <span v-if="piStatusError" class="text-xs text-warning">Status unavailable</span>
                <span
                  v-else
                  class="text-xs"
                  :class="piServerRunning ? 'text-success' : 'text-text-muted'"
                >
                  {{ piServerRunning ? 'Running' : 'Stopped' }}
                </span>
              </div>
              <span v-if="piServerStatusMsg" class="text-xs text-text-muted">{{
                piServerStatusMsg
              }}</span>
            </div>
            <div class="flex gap-2 pt-1">
              <button
                v-if="!piStatusError && !piServerRunning"
                :disabled="piLoading"
                class="rounded-md bg-brand px-4 py-2 text-sm font-semibold text-brand-text transition-colors hover:bg-brand-hover disabled:opacity-50"
                @click="handlePiStart"
              >
                {{ piLoading ? 'Starting...' : 'Start Server' }}
              </button>
              <button
                v-else-if="!piStatusError"
                :disabled="piLoading"
                class="rounded-md border border-danger/30 bg-danger-bg px-4 py-2 text-sm font-medium text-danger transition-colors hover:bg-danger/10 disabled:opacity-50"
                @click="handlePiStop"
              >
                {{ piLoading ? 'Stopping...' : 'Stop Server' }}
              </button>
              <button
                v-if="!piStatusError"
                :disabled="piLoading"
                class="rounded-md border border-border-subtle px-4 py-2 text-sm font-medium text-text-muted transition-colors hover:border-border-default hover:text-text-primary disabled:opacity-50"
                @click="refreshPiStatus"
              >
                Refresh
              </button>
            </div>
            <div class="flex gap-2 pt-1">
              <button
                :disabled="testingPi"
                class="rounded-md bg-brand px-4 py-2 text-sm font-semibold text-brand-text transition-colors hover:bg-brand-hover disabled:opacity-50"
                @click="testPiConnection"
              >
                {{ testingPi ? 'Testing...' : 'Test Inference' }}
              </button>
              <span
                v-if="piTestResult"
                class="text-xs"
                :class="piTestSuccess ? 'text-success' : 'text-danger'"
              >
                {{ piTestResult }}
              </span>
            </div>
          </div>

          <!-- Pi CLI agent -->
          <div v-show="llmSettings.llmBackend === 'pi_agent'" class="space-y-4 mb-6">
            <h3 class="text-sm font-medium text-text-secondary">Pi CLI Agent</h3>
            <p class="text-xs text-text-muted">
              Uses the locally installed Pi agent in print-only, no-tools mode. Authenticate Pi
              separately with <code>pi /login</code> or its provider setup.
            </p>
            <div class="flex gap-2">
              <input
                v-model="llmSettings.piAgentProvider"
                placeholder="openai-codex"
                class="flex-1 rounded-md border border-border-subtle bg-bg-primary px-3 py-2 text-sm text-text-primary placeholder:text-text-muted focus:border-brand focus:outline-none"
              />
              <span class="self-center text-xs text-text-muted">Provider</span>
            </div>
            <div class="flex gap-2">
              <input
                v-model="llmSettings.piAgentModel"
                placeholder="gpt-5.6-luna"
                class="flex-1 rounded-md border border-border-subtle bg-bg-primary px-3 py-2 text-sm text-text-primary placeholder:text-text-muted focus:border-brand focus:outline-none"
              />
              <span class="self-center text-xs text-text-muted">Model</span>
            </div>
            <div class="flex items-center justify-between">
              <p class="text-xs text-text-muted">
                Inference runs through the local <code>pi</code> executable.
              </p>
              <button
                :disabled="testingPiAgent"
                class="rounded-md bg-brand px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-brand/90 disabled:cursor-not-allowed disabled:opacity-50"
                @click="testPiAgentConnection"
              >
                {{ testingPiAgent ? 'Testing...' : 'Test' }}
              </button>
            </div>
            <div
              v-if="piAgentTestResult"
              class="text-xs"
              :class="piAgentTestSuccess ? 'text-success' : 'text-danger'"
            >
              {{ piAgentTestResult }}
            </div>
          </div>

          <!-- OpenAI-compatible LLM (Super Analysis) -->
          <div v-show="llmSettings.llmBackend === 'openai'" class="space-y-4 mb-6">
            <h3 class="text-sm font-medium text-text-secondary">OpenAI-compatible LLM</h3>
            <div class="flex gap-2">
              <input
                v-model="llmSettings.openaiBaseUrl"
                placeholder="https://api.openai.com/v1"
                class="flex-1 rounded-md border border-border-subtle bg-bg-primary px-3 py-2 text-sm text-text-primary placeholder:text-text-muted focus:border-brand focus:outline-none"
              />
            </div>

            <div class="flex gap-2">
              <input
                v-model="llmSettings.openaiApiKey"
                type="password"
                autocomplete="new-password"
                placeholder="Leave blank to keep the configured key"
                class="flex-1 rounded-md border border-border-subtle bg-bg-primary px-3 py-2 text-sm text-text-primary placeholder:text-text-muted focus:border-brand focus:outline-none"
              />
              <span class="self-center text-xs text-text-muted">API Key</span>
            </div>

            <div class="flex gap-2">
              <input
                v-model="llmSettings.openaiModel"
                placeholder="gpt-4o"
                class="flex-1 rounded-md border border-border-subtle bg-bg-primary px-3 py-2 text-sm text-text-primary placeholder:text-text-muted focus:border-brand focus:outline-none"
              />
              <span class="self-center text-xs text-text-muted">Model</span>
            </div>
            <div class="flex items-center justify-between">
              <p class="text-xs text-text-muted">
                External OpenAI-compatible endpoint for sentiment analysis. No server management
                needed.
              </p>
              <button
                :disabled="testingOpenai"
                class="rounded-md bg-brand px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-brand/90 disabled:cursor-not-allowed disabled:opacity-50"
                @click="testOpenAiConnection"
              >
                {{ testingOpenai ? 'Testing...' : 'Test' }}
              </button>
            </div>

            <div
              v-if="openaiTestResult"
              class="text-xs"
              :class="openaiTestSuccess ? 'text-success' : 'text-danger'"
            >
              {{ openaiTestResult }}
            </div>
          </div>

          <!-- Ollama (Local LLM server) -->
          <div v-show="llmSettings.llmBackend === 'ollama'" class="space-y-4 mb-6">
            <h3 class="text-sm font-medium text-text-secondary">Ollama</h3>
            <div class="flex gap-2">
              <input
                v-model="llmSettings.ollamaBaseUrl"
                placeholder="http://localhost:11434/v1"
                class="flex-1 rounded-md border border-border-subtle bg-bg-primary px-3 py-2 text-sm text-text-primary placeholder:text-text-muted focus:border-brand focus:outline-none"
              />
            </div>

            <div class="flex gap-2">
              <input
                v-model="llmSettings.ollamaModel"
                placeholder="qwen3:4b"
                class="flex-1 rounded-md border border-border-subtle bg-bg-primary px-3 py-2 text-sm text-text-primary placeholder:text-text-muted focus:border-brand focus:outline-none"
              />
              <span class="self-center text-xs text-text-muted">Model</span>
            </div>

            <div class="flex gap-2">
              <input
                v-model="llmSettings.ollamaApiKey"
                type="password"
                autocomplete="new-password"
                placeholder="Optional; leave blank to keep the configured key"
                class="flex-1 rounded-md border border-border-subtle bg-bg-primary px-3 py-2 text-sm text-text-primary placeholder:text-text-muted focus:border-brand focus:outline-none"
              />
              <span class="self-center text-xs text-text-muted">API Key (optional)</span>
            </div>
            <div class="flex items-center justify-between">
              <p class="text-xs text-text-muted">
                Local Ollama server for sentiment analysis. Runs entirely on your machine.
              </p>
              <button
                :disabled="testingOllama"
                class="rounded-md bg-brand px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-brand/90 disabled:cursor-not-allowed disabled:opacity-50"
                @click="testOllamaConnection"
              >
                {{ testingOllama ? 'Testing...' : 'Test' }}
              </button>
            </div>

            <div
              v-if="ollamaTestResult"
              class="text-xs"
              :class="ollamaTestSuccess ? 'text-success' : 'text-danger'"
            >
              {{ ollamaTestResult }}
            </div>
          </div>

          <!-- Laya (Remote OpenAI-compatible LLM) -->
          <div v-show="llmSettings.llmBackend === 'laya'" class="space-y-4 mb-6">
            <h3 class="text-sm font-medium text-text-secondary">Laya Backend</h3>
            <p class="text-xs text-text-muted">
              Remote Laya backend (OpenAI-compatible), configured from the dashboard. No server
              management needed.
            </p>
            <div class="flex gap-2">
              <input
                v-model="llmSettings.layaBaseUrl"
                placeholder="https://laya.example/v1"
                class="flex-1 rounded-md border border-border-subtle bg-bg-primary px-3 py-2 text-sm text-text-primary placeholder:text-text-muted focus:border-brand focus:outline-none"
              />
            </div>

            <div class="flex gap-2">
              <input
                v-model="llmSettings.layaModel"
                placeholder="Ornith-1.5-35B-A3B-AWQ"
                class="flex-1 rounded-md border border-border-subtle bg-bg-primary px-3 py-2 text-sm text-text-primary placeholder:text-text-muted focus:border-brand focus:outline-none"
              />
              <span class="self-center text-xs text-text-muted">Model</span>
            </div>
          </div>

          <!-- PDF Extraction -->
          <div class="space-y-4 mb-6">
            <h3 class="text-sm font-medium text-text-secondary">PDF Extraction (Pi 5)</h3>
            <div class="flex gap-2">
              <input
                v-model="llmSettings.pdfBaseUrl"
                placeholder="http://pi5-ip:8080"
                class="flex-1 rounded-md border border-border-subtle bg-bg-primary px-3 py-2 text-sm text-text-primary placeholder:text-text-muted focus:border-brand focus:outline-none"
              />
              <button
                :disabled="testingPdf"
                class="rounded-md border border-brand bg-brand-subtle px-4 py-2 text-sm font-medium text-brand transition-colors hover:bg-brand/20 disabled:opacity-50"
                @click="testPdfExtraction"
              >
                {{ testingPdf ? 'Testing...' : 'Test' }}
              </button>
            </div>

            <div class="flex gap-2">
              <input
                v-model="llmSettings.pdfModel"
                placeholder="gemma-4-E2B"
                class="flex-1 rounded-md border border-border-subtle bg-bg-primary px-3 py-2 text-sm text-text-primary placeholder:text-text-muted focus:border-brand focus:outline-none"
              />
              <span class="self-center text-xs text-text-muted">Model name</span>
            </div>
          </div>

          <!-- Discord Configuration -->
          <div class="space-y-4">
            <h3 class="text-sm font-medium text-text-secondary">Discord Notifications</h3>
            <div
              class="flex items-center justify-between rounded-lg border border-border-subtle p-3"
            >
              <span class="text-sm">Enable Discord</span>
              <label class="relative inline-flex items-center cursor-pointer">
                <input v-model="discordSettings.enabled" type="checkbox" class="sr-only peer" />
                <div
                  class="w-9 h-5 bg-gray-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-brand"
                />
              </label>
            </div>
            <div class="flex gap-2">
              <input
                v-model="discordSettings.webhookUrl"
                placeholder="https://discord.com/api/webhooks/..."
                class="flex-1 rounded-md border border-border-subtle bg-bg-primary px-3 py-2 text-sm text-text-primary placeholder:text-text-muted focus:border-brand focus:outline-none"
              />
              <button
                :disabled="testingDiscord"
                class="rounded-md border border-brand bg-brand-subtle px-4 py-2 text-sm font-medium text-brand transition-colors hover:bg-brand/20 disabled:opacity-50"
                @click="testDiscordWebhook"
              >
                {{ testingDiscord ? 'Testing...' : 'Test' }}
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Trading Tab -->
      <div v-show="activeTab === 'trading'">
        <div class="settings-panel trading-card card-panel p-5 sm:p-6">
          <div class="settings-panel-heading">
            <div>
              <p class="settings-section-kicker">Execution guardrails</p>
              <h2 class="mt-1 text-lg font-semibold text-text-primary">Trading Configuration</h2>
              <p class="mt-1 text-sm text-text-muted">
                Set the rules that keep every paper trade within your plan.
              </p>
            </div>
            <span class="settings-panel-mark">03</span>
          </div>

          <div class="settings-mode-row mt-6 rounded-xl border border-border-subtle p-4">
            <div>
              <p class="text-sm font-semibold text-text-primary">Trading mode</p>
              <p class="mt-1 text-xs text-text-muted">
                Live trading stays opt-in until you deliberately switch modes.
              </p>
            </div>
            <select
              v-model="settings.tradingConfig.mode"
              aria-label="Trading mode"
              class="settings-input mt-3 w-full sm:mt-0 sm:w-52"
            >
              <option value="paper">Paper Trading</option>
              <option value="live">Live Trading</option>
            </select>
          </div>

          <div class="mt-5 grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
            <label class="settings-field rounded-xl border border-border-subtle p-4">
              <span class="settings-field-label">Initial Capital</span>
              <span class="settings-field-help">Paper-trading starting balance</span>
              <span class="settings-input-wrap">
                <span>Rs.</span>
                <input
                  v-model.number="settings.tradingConfig.initialCapital"
                  type="number"
                  min="1"
                  step="1000"
                  class="settings-input"
                />
              </span>
            </label>
            <label class="settings-field rounded-xl border border-border-subtle p-4">
              <span class="settings-field-label">Max Position Size</span>
              <span class="settings-field-help">Portfolio allocation limit</span>
              <span class="settings-input-wrap">
                <input
                  v-model.number="settings.tradingConfig.maxPositionSize"
                  type="number"
                  min="1"
                  max="100"
                  class="settings-input"
                />
                <span>%</span>
              </span>
            </label>
            <label class="settings-field rounded-xl border border-border-subtle p-4">
              <span class="settings-field-label">Stop Loss</span>
              <span class="settings-field-help">Exit when risk threshold hits</span>
              <span class="settings-input-wrap">
                <input
                  v-model.number="settings.tradingConfig.stopLoss"
                  type="number"
                  min="1"
                  max="50"
                  class="settings-input"
                />
                <span>%</span>
              </span>
            </label>
            <label class="settings-field rounded-xl border border-border-subtle p-4">
              <span class="settings-field-label">Take Profit</span>
              <span class="settings-field-help">Target gain before exit</span>
              <span class="settings-input-wrap">
                <input
                  v-model.number="settings.tradingConfig.takeProfit"
                  type="number"
                  min="1"
                  max="200"
                  class="settings-input"
                />
                <span>%</span>
              </span>
            </label>
          </div>
          <p class="mt-4 text-xs text-text-muted">
            Initial capital changes apply after the API restarts; the current paper portfolio is not
            reset automatically.
          </p>
        </div>
      </div>

      <!-- Scanning Tab -->
      <div v-show="activeTab === 'scanning'">
        <div class="settings-panel card-panel p-5 sm:p-6">
          <div class="settings-panel-heading">
            <div>
              <p class="settings-section-kicker">Research throughput</p>
              <h2 class="mt-1 text-lg font-semibold text-text-primary">Candidate Explorer</h2>
              <p class="mt-1 max-w-xl text-sm leading-6 text-text-muted">
                Choose how many symbols can be processed at once during the next scan. Higher values
                finish faster but can increase provider load.
              </p>
            </div>
            <span class="settings-panel-mark">05</span>
          </div>

          <label
            class="settings-field mt-6 block max-w-md rounded-xl border border-border-subtle p-4"
          >
            <span class="settings-field-label">Parallel workers</span>
            <span class="settings-field-help">Allowed range: 1–12. Default: 3.</span>
            <span class="settings-input-wrap mt-3">
              <input
                v-model.number="scanningConfig.maxConcurrent"
                type="number"
                min="1"
                max="12"
                step="1"
                aria-label="Candidate Explorer parallel workers"
                class="settings-input"
              />
              <span>workers</span>
            </span>
          </label>
          <p class="mt-4 text-xs leading-5 text-text-muted">
            This setting is applied when a new scan starts; an active scan keeps its original worker
            limit.
          </p>
        </div>
      </div>

      <!-- Health Tab -->
      <div v-show="activeTab === 'health'">
        <div class="settings-panel health-card card-panel p-5 sm:p-6">
          <div class="settings-panel-heading">
            <div>
              <p class="settings-section-kicker">Operational pulse</p>
              <h2 class="mt-1 text-lg font-semibold text-text-primary">System Health</h2>
              <p class="mt-1 text-sm text-text-muted">
                A quick read on the services supporting your workspace.
              </p>
            </div>
            <span class="settings-panel-mark">04</span>
          </div>
          <div v-if="healthStatus" class="settings-health-list mt-6 space-y-2">
            <div
              v-for="(comp, key) in healthStatus.components"
              :key="key"
              class="settings-health-row flex items-center justify-between rounded-xl border border-border-subtle/50 p-4"
            >
              <div class="flex items-center gap-3">
                <span
                  class="settings-health-icon"
                  :class="healthDot(comp.status)"
                  aria-hidden="true"
                />
                <div>
                  <span class="block text-sm font-semibold capitalize text-text-primary">{{
                    key
                  }}</span>
                  <span class="mt-0.5 block text-xs text-text-muted">Service availability</span>
                </div>
              </div>
              <span
                class="inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium"
                :class="healthColor(comp.status)"
              >
                {{ comp.status }}
              </span>
            </div>
          </div>
          <div
            v-else-if="healthStatusError"
            role="status"
            class="settings-health-empty mt-6 rounded-xl border border-warning/30 bg-warning-bg p-5 text-sm text-warning"
          >
            <span class="font-semibold">Status unavailable.</span> We couldn't confirm every service
            right now.
          </div>
          <div
            v-else
            class="flex items-center justify-center rounded-xl border border-border-subtle p-8"
          >
            <LoadingSpinner :message="'Checking system...'" :small="true" />
          </div>
        </div>
      </div>
    </div>

    <!-- Toast notifications -->
    <Transition
      enter-active-class="transition ease-out duration-200"
      enter-from-class="opacity-0 translate-y-2"
      enter-to-class="opacity-100 translate-y-0"
      leave-active-class="transition ease-in duration-150"
      leave-from-class="opacity-100 translate-y-0"
      leave-to-class="opacity-0 translate-y-2"
    >
      <Toast v-if="toastVisible" :message="toastMessage" :type="toastType" :duration="4000" />
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  getFyersLoginUrl,
  getFyersStatus,
  fyersAuthCode,
  fyersLogout,
  testDiscordWebhook as apiTestDiscordWebhook,
  testPiConnection as apiTestPiConnection,
  testOpenAiConnection as apiTestOpenAiConnection,
  testOllamaConnection as apiTestOllamaConnection,
  testPiAgentConnection as apiTestPiAgentConnection,
  startPiServer as apiStartPiServer,
  stopPiServer as apiStopPiServer,
  getPiServerStatus as apiGetPiServerStatus,
} from '../api/client'
import type { FyersStatus, HealthStatus } from '../api/types'
import { formatAppError } from '../errors/appError'
import LoadingSpinner from '../components/LoadingSpinner.vue'
import Toast from '../components/Toast.vue'
import {
  getSettings,
  loadSettings,
  saveSettings,
  saveLlmSettings,
  saveDiscordSettings,
} from '../stores/settings'

const activeTab = ref('broker')
const settings = getSettings()
const llmSettings = settings.llmSettings
const discordSettings = settings.discordSettings
const scanningConfig = settings.scanningConfig ?? { maxConcurrent: 3 }

function confirmed<T>(value: T | { success: boolean; data?: T }): T | undefined {
  if (typeof value === 'object' && value !== null && 'data' in value) {
    return value.success ? value.data : undefined
  }
  return value as T
}
const fyersStatus = ref<FyersStatus | null>(null)
const fyersStatusError = ref(false)
const fyersConnected = computed(
  () => !fyersStatusError.value && (fyersStatus.value?.connected ?? false)
)
const healthStatus = ref<HealthStatus | null>(null)
const healthStatusError = ref(false)
const authResultBanner = ref<'success' | 'error' | null>(null)

const authing = ref(false)
const showAuthCodeInput = ref(false)
const authCodeInput = ref('')
const testingPdf = ref(false)
const testingDiscord = ref(false)
const testingPi = ref(false)
const testingOpenai = ref(false)
const testingOllama = ref(false)
const testingPiAgent = ref(false)
const piTestResult = ref('')
const piTestSuccess = ref(false)
const piServerRunning = ref(false)
const piServerStatusMsg = ref('')
const piLoading = ref(false)
const piStatusError = ref(false)
const unconfirmedDefaults = ref(false)
const openaiTestResult = ref('')
const openaiTestSuccess = ref(false)
const ollamaTestResult = ref('')
const ollamaTestSuccess = ref(false)
const piAgentTestResult = ref('')
const piAgentTestSuccess = ref(false)
const saving = ref(false)
const saved = ref(false)
const toastMessage = ref('')
const toastType = ref<'success' | 'error' | 'warning' | 'info'>('info')
const toastVisible = ref(false)
let pollTimer: number | null = null

const brokers = [
  { value: 'fyers' as const, label: 'Fyers' },
  { value: 'upstox' as const, label: 'Upstox' },
  { value: 'yahoo' as const, label: 'Yahoo Finance' },
  { value: 'none' as const, label: 'None (Read Only)' },
]

const llmBackends = [
  { value: 'local' as const, label: 'Local' },
  { value: 'pi_ssh' as const, label: 'Pi SSH' },
  { value: 'openai' as const, label: 'OpenAI' },
  { value: 'ollama' as const, label: 'Ollama' },
  { value: 'laya' as const, label: 'Laya' },
  { value: 'pi_agent' as const, label: 'Pi Agent' },
]

const healthColor = (status: string) => {
  if (status === 'UP') return 'bg-success-bg text-success'
  if (status === 'DOWN') return 'bg-danger-bg text-danger'
  if (status === 'DEGRADED') return 'bg-warning-bg text-warning'
  return 'bg-bg-hover text-text-muted'
}

const healthDot = (status: string) => {
  if (status === 'UP') return 'bg-success'
  if (status === 'DOWN') return 'bg-danger'
  if (status === 'DEGRADED') return 'bg-warning'
  return 'bg-text-muted'
}

const refreshFyersStatus = async () => {
  if (settings.selectedBroker !== 'fyers') {
    fyersStatus.value = null
    fyersStatusError.value = false
    return
  }
  fyersStatusError.value = false
  try {
    const res = await getFyersStatus()
    const data = confirmed(res)
    if (data) fyersStatus.value = data
    else fyersStatusError.value = true
  } catch {
    fyersStatusError.value = true
  }
}

const pollFyersStatus = () => {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
  pollTimer = window.setInterval(async () => {
    try {
      const res = await getFyersStatus()
      const data = confirmed(res)
      if (data?.connected) {
        if (pollTimer) {
          clearInterval(pollTimer)
          pollTimer = null
        }
        authResultBanner.value = 'success'
        showAuthCodeInput.value = false
        fyersStatus.value = data
      }
    } catch {
      // ignore polling errors
    }
  }, 2000) as unknown as number
}

const refreshHealth = async () => {
  const { getHealthStatus } = await import('../api/client')
  healthStatusError.value = false
  try {
    const res = await getHealthStatus()
    const legacy = res as unknown as { success?: boolean; data?: HealthStatus }
    const data = legacy.success !== undefined ? (legacy.success ? legacy.data : undefined) : res
    if (data) healthStatus.value = data
    else healthStatusError.value = true
  } catch {
    healthStatusError.value = true
  }
}

const startFyersAuth = async () => {
  authing.value = true
  try {
    const res = await getFyersLoginUrl()
    const data = confirmed(res)
    if (!data) throw new Error('Failed to get login URL')

    const popup = window.open(
      data.url,
      'fyers-auth',
      'width=600,height=700,left=' +
        Math.round(window.screen.width / 2 - 300) +
        ',top=' +
        Math.round(window.screen.height / 2 - 350)
    )
    if (!popup) {
      authResultBanner.value = 'error'
      authing.value = false
      throw new Error('Popup blocked. Please allow popups for this site.')
    }

    pollFyersStatus()

    popup.addEventListener('load', () => {
      if (pollTimer) {
        clearInterval(pollTimer)
        pollTimer = null
      }
    })
  } catch {
    authResultBanner.value = 'error'
  } finally {
    authing.value = false
  }
}

const submitAuthCode = async () => {
  if (!authCodeInput.value.trim()) return
  authing.value = true
  try {
    const res = await fyersAuthCode(authCodeInput.value.trim())
    const data = confirmed(res)
    if (!data) throw new Error('Auth failed')
    fyersStatus.value = data
    showAuthCodeInput.value = false
    authCodeInput.value = ''
  } catch {
    authResultBanner.value = 'error'
    toastMessage.value = 'Fyers authentication failed. Please try again.'
    toastType.value = 'error'
    toastVisible.value = true
    setTimeout(() => {
      toastVisible.value = false
    }, 4000)
  } finally {
    authing.value = false
  }
}

const disconnectFyers = async () => {
  const res = await fyersLogout()
  const data = confirmed(res)
  if (data) fyersStatus.value = data
}

const handleSave = async () => {
  saving.value = true
  saved.value = false
  try {
    const ok = await saveSettings()
    if (ok) {
      saved.value = true
      toastMessage.value = 'Settings saved!'
      toastType.value = 'success'
      toastVisible.value = true
      setTimeout(() => {
        saved.value = false
      }, 2000)
      setTimeout(() => {
        toastVisible.value = false
      }, 4000)
    } else {
      toastMessage.value = 'Failed to save settings'
      toastType.value = 'error'
      toastVisible.value = true
      setTimeout(() => {
        toastVisible.value = false
      }, 4000)
    }
  } catch (err: unknown) {
    toastMessage.value = formatAppError(err, {
      title: 'Settings could not be saved',
      operation: 'mutation',
      refreshLabel: 'Refresh settings',
    }).message
    toastType.value = 'error'
    toastVisible.value = true
    setTimeout(() => {
      toastVisible.value = false
    }, 4000)
  } finally {
    saving.value = false
  }
}

const testPdfExtraction = async () => {
  testingPdf.value = true
  try {
    const ok = await saveLlmSettings()
    if (ok) {
      toastMessage.value = 'PDF extraction settings saved.'
      toastType.value = 'success'
      toastVisible.value = true
      setTimeout(() => {
        toastVisible.value = false
      }, 4000)
    } else {
      toastMessage.value = 'Failed to save PDF settings.'
      toastType.value = 'error'
      toastVisible.value = true
      setTimeout(() => {
        toastVisible.value = false
      }, 4000)
    }
  } catch (err: unknown) {
    toastMessage.value = formatAppError(err, {
      title: 'PDF settings could not be saved',
      operation: 'mutation',
    }).message
    toastType.value = 'error'
    toastVisible.value = true
    setTimeout(() => {
      toastVisible.value = false
    }, 4000)
  } finally {
    testingPdf.value = false
  }
}

const testDiscordWebhook = async () => {
  const ok = await saveDiscordSettings()
  if (!ok) {
    toastMessage.value = 'Failed to save Discord settings.'
    toastType.value = 'error'
    toastVisible.value = true
    setTimeout(() => {
      toastVisible.value = false
    }, 4000)
    return
  }
  testingDiscord.value = true
  try {
    const result = confirmed(await apiTestDiscordWebhook())
    if (result?.success) {
      toastMessage.value = 'Discord webhook test successful!'
      toastType.value = 'success'
      toastVisible.value = true
      setTimeout(() => {
        toastVisible.value = false
      }, 4000)
    } else {
      toastMessage.value = 'Discord webhook test failed'
      toastType.value = 'error'
      toastVisible.value = true
      setTimeout(() => {
        toastVisible.value = false
      }, 4000)
    }
  } catch (err: unknown) {
    toastMessage.value = formatAppError(err, {
      title: 'Discord webhook test failed',
      operation: 'mutation',
    }).message
    toastType.value = 'error'
    toastVisible.value = true
    setTimeout(() => {
      toastVisible.value = false
    }, 4000)
  } finally {
    testingDiscord.value = false
  }
}

const refreshPiStatus = async () => {
  piStatusError.value = false
  try {
    const status = confirmed(await apiGetPiServerStatus())
    if (status) {
      piServerRunning.value = status.running ?? false
      piServerStatusMsg.value = status.message ?? ''
    }
  } catch {
    piStatusError.value = true
  }
}

const handlePiStart = async () => {
  piLoading.value = true
  try {
    const result = confirmed(await apiStartPiServer())
    if (result) {
      piServerRunning.value = result.running ?? false
      piServerStatusMsg.value = result.message ?? ''
      toastMessage.value = result.message ?? ''
      toastType.value = result.success ? 'success' : 'error'
      toastVisible.value = true
      setTimeout(() => {
        toastVisible.value = false
      }, 4000)
    }
  } catch (err: unknown) {
    toastMessage.value = formatAppError(err, {
      title: 'Pi server could not start',
      operation: 'mutation',
    }).message
    toastType.value = 'error'
    toastVisible.value = true
    setTimeout(() => {
      toastVisible.value = false
    }, 4000)
  } finally {
    piLoading.value = false
  }
}

const handlePiStop = async () => {
  piLoading.value = true
  try {
    const result = confirmed(await apiStopPiServer())
    if (result) {
      piServerRunning.value = result.running ?? false
      piServerStatusMsg.value = result.message ?? ''
      toastMessage.value = result.message ?? ''
      toastType.value = result.success ? 'success' : 'error'
      toastVisible.value = true
      setTimeout(() => {
        toastVisible.value = false
      }, 4000)
    }
  } catch (err: unknown) {
    toastMessage.value = formatAppError(err, {
      title: 'Pi server could not stop',
      operation: 'mutation',
    }).message
    toastType.value = 'error'
    toastVisible.value = true
    setTimeout(() => {
      toastVisible.value = false
    }, 4000)
  } finally {
    piLoading.value = false
  }
}

const testPiConnection = async () => {
  testingPi.value = true
  piTestResult.value = ''
  piTestSuccess.value = false
  try {
    const result = confirmed(await apiTestPiConnection())
    if (result) {
      piTestResult.value = result.message ?? (result.success ? 'Connected!' : 'Failed to start')
      piTestSuccess.value = result.success
      if (result.success) {
        toastMessage.value = 'Pi SSH connection successful — llama-server started on Pi'
        toastType.value = 'success'
        toastVisible.value = true
        setTimeout(() => {
          toastVisible.value = false
        }, 4000)
        refreshPiStatus()
      } else {
        toastMessage.value = 'Pi connected but llama-server failed to start'
        toastType.value = 'warning'
        toastVisible.value = true
        setTimeout(() => {
          toastVisible.value = false
        }, 4000)
      }
    } else {
      piTestResult.value = 'Test failed'
      piTestSuccess.value = false
      toastMessage.value = piTestResult.value
      toastType.value = 'error'
      toastVisible.value = true
      setTimeout(() => {
        toastVisible.value = false
      }, 4000)
    }
  } catch (err: unknown) {
    piTestResult.value = formatAppError(err, {
      title: 'Pi inference test failed',
      operation: 'mutation',
    }).message
    piTestSuccess.value = false
    toastMessage.value = piTestResult.value
    toastType.value = 'error'
    toastVisible.value = true
    setTimeout(() => {
      toastVisible.value = false
    }, 4000)
  } finally {
    testingPi.value = false
  }
}

const testOpenAiConnection = async () => {
  testingOpenai.value = true
  openaiTestResult.value = ''
  openaiTestSuccess.value = false
  try {
    const result = confirmed(await apiTestOpenAiConnection())
    if (result) {
      openaiTestResult.value = result.message ?? (result.success ? 'Connected!' : 'Failed')
      openaiTestSuccess.value = result.success
      if (result.success) {
        toastMessage.value = 'OpenAI-compatible LLM responded successfully'
        toastType.value = 'success'
        toastVisible.value = true
        setTimeout(() => {
          toastVisible.value = false
        }, 4000)
      } else {
        toastMessage.value = 'LLM responded but unexpected output'
        toastType.value = 'warning'
        toastVisible.value = true
        setTimeout(() => {
          toastVisible.value = false
        }, 4000)
      }
    } else {
      openaiTestResult.value = 'Test failed'
      openaiTestSuccess.value = false
      toastMessage.value = openaiTestResult.value
      toastType.value = 'error'
      toastVisible.value = true
      setTimeout(() => {
        toastVisible.value = false
      }, 4000)
    }
  } catch (err: unknown) {
    openaiTestResult.value = formatAppError(err, {
      title: 'OpenAI connection test failed',
      operation: 'mutation',
    }).message
    openaiTestSuccess.value = false
    toastMessage.value = openaiTestResult.value
    toastType.value = 'error'
    toastVisible.value = true
    setTimeout(() => {
      toastVisible.value = false
    }, 4000)
  } finally {
    testingOpenai.value = false
  }
}

const testPiAgentConnection = async () => {
  testingPiAgent.value = true
  piAgentTestResult.value = ''
  piAgentTestSuccess.value = false
  try {
    const result = confirmed(await apiTestPiAgentConnection())
    if (result) {
      piAgentTestResult.value = result.message ?? (result.success ? 'Connected!' : 'Failed')
      piAgentTestSuccess.value = result.success
      toastMessage.value =
        result.message ??
        (result.success ? 'Pi agent responded successfully' : 'Pi agent test failed')
      toastType.value = result.success ? 'success' : 'error'
      toastVisible.value = true
      setTimeout(() => {
        toastVisible.value = false
      }, 4000)
    }
  } catch (err: unknown) {
    piAgentTestResult.value = formatAppError(err, {
      title: 'Pi agent test failed',
      operation: 'mutation',
    }).message
    piAgentTestSuccess.value = false
    toastMessage.value = piAgentTestResult.value
    toastType.value = 'error'
    toastVisible.value = true
    setTimeout(() => {
      toastVisible.value = false
    }, 4000)
  } finally {
    testingPiAgent.value = false
  }
}

const testOllamaConnection = async () => {
  testingOllama.value = true
  ollamaTestResult.value = ''
  ollamaTestSuccess.value = false
  try {
    const result = confirmed(await apiTestOllamaConnection())
    if (result) {
      ollamaTestResult.value = result.message ?? (result.success ? 'Connected!' : 'Failed')
      ollamaTestSuccess.value = result.success
      if (result.success) {
        toastMessage.value = 'Ollama responded successfully'
        toastType.value = 'success'
        toastVisible.value = true
        setTimeout(() => {
          toastVisible.value = false
        }, 4000)
      } else {
        toastMessage.value = 'Ollama responded but unexpected output'
        toastType.value = 'warning'
        toastVisible.value = true
        setTimeout(() => {
          toastVisible.value = false
        }, 4000)
      }
    } else {
      ollamaTestResult.value = 'Test failed'
      ollamaTestSuccess.value = false
      toastMessage.value = ollamaTestResult.value
      toastType.value = 'error'
      toastVisible.value = true
      setTimeout(() => {
        toastVisible.value = false
      }, 4000)
    }
  } catch (err: unknown) {
    ollamaTestResult.value = formatAppError(err, {
      title: 'Ollama connection test failed',
      operation: 'mutation',
    }).message
    ollamaTestSuccess.value = false
    toastMessage.value = ollamaTestResult.value
    toastType.value = 'error'
    toastVisible.value = true
    setTimeout(() => {
      toastVisible.value = false
    }, 4000)
  } finally {
    testingOllama.value = false
  }
}

const handleMessage = (event: MessageEvent) => {
  if (event.data?.type === 'fyers_auth_success') {
    authResultBanner.value = 'success'
    showAuthCodeInput.value = false
    if (pollTimer) {
      clearInterval(pollTimer)
      pollTimer = null
    }
    refreshFyersStatus()
  } else if (event.data?.type === 'fyers_auth_error') {
    authResultBanner.value = 'error'
    authing.value = false
    showAuthCodeInput.value = false
    if (pollTimer) {
      clearInterval(pollTimer)
      pollTimer = null
    }
  }
}

onMounted(async () => {
  const route = useRoute()
  const router = useRouter()
  const authParam = (route?.query?.auth as string) ?? null
  if (authParam === 'success' || authParam === 'error') {
    authResultBanner.value = authParam
    router.replace({ query: {} })
  }
  const failedSections = await loadSettings()
  unconfirmedDefaults.value = failedSections.length > 0
  if (failedSections.length > 0) {
    toastMessage.value = `Failed to load: ${failedSections.join(', ')}. Showing defaults.`
    toastType.value = 'warning'
    toastVisible.value = true
    setTimeout(() => {
      toastVisible.value = false
    }, 6000)
  }
  await refreshFyersStatus()
  refreshHealth()
  refreshPiStatus()
  window.addEventListener('message', handleMessage)
})

onUnmounted(() => {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
  window.removeEventListener('message', handleMessage)
})
</script>

<style scoped>
.settings-shell {
  --settings-ease: cubic-bezier(0.23, 1, 0.32, 1);
}

.settings-header {
  position: relative;
  overflow: hidden;
  background:
    radial-gradient(
      circle at 100% 0%,
      color-mix(in srgb, var(--color-brand) 12%, transparent),
      transparent 34%
    ),
    linear-gradient(
      135deg,
      color-mix(in srgb, var(--color-bg-surface) 96%, white),
      var(--color-bg-primary)
    );
}

.settings-header::after {
  position: absolute;
  right: 2rem;
  bottom: -4rem;
  width: 12rem;
  height: 12rem;
  border: 1px solid color-mix(in srgb, var(--color-brand) 18%, transparent);
  border-radius: 999px;
  content: '';
  pointer-events: none;
}

.settings-kicker,
.settings-card-label,
.settings-tab-index,
.settings-tab-detail {
  color: var(--color-text-muted);
  font-size: 0.68rem;
  font-weight: 700;
  letter-spacing: 0.12em;
  line-height: 1;
  text-transform: uppercase;
}

.settings-kicker span {
  color: var(--color-brand);
  margin: 0 0.35rem;
}

.settings-save,
.settings-tab,
.settings-status-card {
  position: relative;
  transition:
    transform 160ms var(--settings-ease),
    border-color 160ms ease,
    background-color 160ms ease,
    color 160ms ease;
}

.settings-save:active,
.settings-tab:active {
  transform: scale(0.97);
}

.settings-status-card {
  display: flex;
  min-height: 5.25rem;
  flex-direction: column;
  justify-content: space-between;
  background: color-mix(in srgb, var(--color-bg-surface) 72%, transparent);
}

.settings-status-card:hover {
  border-color: color-mix(in srgb, var(--color-brand) 38%, var(--color-border-subtle));
  transform: translateY(-2px);
}

.settings-card-value {
  color: var(--color-text-primary);
  font-size: 0.95rem;
  font-weight: 650;
}

.settings-panel {
  background: color-mix(in srgb, var(--color-bg-surface) 88%, transparent);
}

.broker-card {
  background:
    radial-gradient(
      circle at 100% 0%,
      color-mix(in srgb, var(--color-brand) 7%, transparent),
      transparent 34%
    ),
    color-mix(in srgb, var(--color-bg-surface) 92%, transparent);
}

.broker-card-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
}

.broker-card-mark {
  color: color-mix(in srgb, var(--color-brand) 70%, var(--color-text-muted));
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.72rem;
  font-weight: 700;
  letter-spacing: 0.08em;
}

.broker-options {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.75rem;
}

.broker-option {
  min-height: 4.75rem;
  text-align: left;
  transition:
    transform 160ms var(--settings-ease),
    border-color 160ms ease,
    background-color 160ms ease,
    color 160ms ease;
}

.broker-option:active {
  transform: scale(0.98);
}

.broker-option-check {
  display: inline-flex;
  width: 1.25rem;
  height: 1.25rem;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  background: var(--color-brand);
  color: var(--color-text-inverse);
  font-size: 0.72rem;
  font-weight: 800;
}

.broker-status {
  background: color-mix(in srgb, var(--color-bg-primary) 42%, transparent);
}

.broker-actions button {
  min-height: 2.75rem;
}

.llm-card {
  background:
    radial-gradient(
      circle at 100% 0%,
      color-mix(in srgb, var(--color-info) 8%, transparent),
      transparent 34%
    ),
    color-mix(in srgb, var(--color-bg-surface) 92%, transparent);
}

.llm-backend-options {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 0.75rem;
}

.llm-backend-option {
  min-height: 3.75rem;
  text-align: left;
  transition:
    transform 160ms var(--settings-ease),
    border-color 160ms ease,
    background-color 160ms ease,
    color 160ms ease;
}

.llm-backend-option:active {
  transform: scale(0.98);
}

.llm-backend-section > .space-y-2 > div {
  border: 1px solid color-mix(in srgb, var(--color-border-subtle) 70%, transparent);
  border-radius: 0.75rem;
  background: color-mix(in srgb, var(--color-bg-primary) 42%, transparent);
  padding: 0.75rem 1rem;
}

.trading-card,
.health-card {
  background:
    radial-gradient(
      circle at 100% 0%,
      color-mix(in srgb, var(--color-brand) 6%, transparent),
      transparent 34%
    ),
    color-mix(in srgb, var(--color-bg-surface) 92%, transparent);
}

.trading-card .settings-mode-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.trading-card .settings-field {
  background: color-mix(in srgb, var(--color-bg-primary) 34%, transparent);
}

.health-card .settings-health-list {
  border-top: 1px solid color-mix(in srgb, var(--color-border-subtle) 70%, transparent);
  padding-top: 1rem;
}

.health-card .settings-health-row {
  min-height: 4.5rem;
}

@media (max-width: 480px) {
  .broker-options,
  .llm-backend-options {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 760px) and (min-width: 481px) {
  .llm-backend-options {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

.settings-panel-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
}

.settings-section-kicker {
  color: var(--color-brand);
  font-size: 0.68rem;
  font-weight: 700;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

.settings-panel-mark {
  color: color-mix(in srgb, var(--color-brand) 70%, var(--color-text-muted));
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.72rem;
  font-weight: 700;
  letter-spacing: 0.08em;
}

.settings-mode-row {
  background: color-mix(in srgb, var(--color-bg-primary) 45%, transparent);
}

.settings-field {
  display: flex;
  min-height: 9.5rem;
  flex-direction: column;
  cursor: text;
  transition:
    border-color 160ms ease,
    transform 160ms var(--settings-ease);
}

.settings-field:focus-within {
  border-color: color-mix(in srgb, var(--color-brand) 60%, var(--color-border-subtle));
  transform: translateY(-2px);
}

.settings-field-label {
  color: var(--color-text-primary);
  font-size: 0.82rem;
  font-weight: 650;
}

.settings-field-help {
  margin-top: 0.35rem;
  color: var(--color-text-muted);
  font-size: 0.68rem;
  line-height: 1.4;
}

.settings-input-wrap {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-top: auto;
  color: var(--color-text-muted);
  font-size: 0.78rem;
}

.settings-input {
  min-height: 2.5rem;
  border: 1px solid var(--color-border-subtle);
  border-radius: 0.65rem;
  background: var(--color-bg-primary);
  color: var(--color-text-primary);
  font-size: 0.86rem;
  outline: none;
  padding: 0.55rem 0.7rem;
  transition:
    border-color 160ms ease,
    box-shadow 160ms ease,
    background-color 160ms ease;
}

.settings-input:focus {
  border-color: var(--color-brand);
  box-shadow: 0 0 0 3px var(--color-brand-subtle);
}

.settings-input-wrap .settings-input {
  width: 100%;
}

.settings-health-row {
  background: color-mix(in srgb, var(--color-bg-primary) 35%, transparent);
  transition:
    border-color 160ms ease,
    transform 160ms var(--settings-ease);
}

.settings-health-row:hover {
  border-color: color-mix(in srgb, var(--color-brand) 30%, var(--color-border-subtle));
  transform: translateX(2px);
}

.settings-health-icon {
  width: 0.6rem;
  height: 0.6rem;
  border-radius: 999px;
  box-shadow: 0 0 0 4px color-mix(in srgb, currentColor 12%, transparent);
}

.settings-health-empty {
  line-height: 1.5;
}

.settings-tab {
  display: grid;
  grid-template-columns: auto 1fr;
  column-gap: 0.55rem;
  row-gap: 0.35rem;
  border: 1px solid transparent;
  border-radius: 0.75rem;
}

.settings-tab-index {
  grid-row: span 2;
  padding-top: 0.1rem;
  color: color-mix(in srgb, var(--color-brand) 72%, var(--color-text-muted));
}

.settings-tab-detail {
  font-size: 0.58rem;
  font-weight: 500;
  letter-spacing: 0.08em;
  text-transform: none;
}

.settings-tab[aria-selected='true'] {
  border-color: color-mix(in srgb, var(--color-brand) 28%, transparent);
  background: color-mix(in srgb, var(--color-brand) 9%, var(--color-bg-surface));
  box-shadow: inset 0 -2px 0 var(--color-brand);
}

@media (max-width: 640px) {
  .settings-tab-detail {
    display: none;
  }

  .settings-tab {
    grid-template-columns: auto 1fr;
  }
}

@media (prefers-reduced-motion: reduce) {
  .settings-save,
  .settings-tab,
  .settings-status-card {
    transition: none;
  }

  .settings-field,
  .settings-health-row,
  .settings-input {
    transition: none;
  }
}
</style>
