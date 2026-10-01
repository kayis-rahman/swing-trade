package com.swingtrade.llm.service;

import com.swingtrade.domain.store.AppSettingsStore;
import org.junit.jupiter.api.Test;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.when;

class LlmServerManagerProviderTest {

    @Test
    void selectsManagerForLocalAndPiBackendsWithoutStartingProcesses() {
        AppSettingsStore settings = mock(AppSettingsStore.class);
        LlamaCppServerManager local = mock(LlamaCppServerManager.class);
        PiLlamaServerManager pi = mock(PiLlamaServerManager.class);

        when(settings.get("llm.backend")).thenReturn(java.util.Optional.of("local"));
        var provider = provider(settings, local, pi);
        assertThat(provider.getManager()).isSameAs(local);

        when(settings.get("llm.backend")).thenReturn(java.util.Optional.of("pi_ssh"));
        assertThat(provider.getManager()).isSameAs(pi);
    }

    @Test
    void returnsNoManagerForBackendsWithoutManagedServers() {
        AppSettingsStore settings = mock(AppSettingsStore.class);
        when(settings.get("llm.backend")).thenReturn(java.util.Optional.of("openai"));
        var provider = provider(settings, mock(LlamaCppServerManager.class), mock(PiLlamaServerManager.class));
        assertThat(provider.getManager()).isNull();

        when(settings.get("llm.backend")).thenReturn(java.util.Optional.of("ollama"));
        assertThat(provider.getManager()).isNull();
    }

    @Test
    void unknownBackendUsesSafeLocalFallback() {
        AppSettingsStore settings = mock(AppSettingsStore.class);
        when(settings.get("llm.backend")).thenReturn(java.util.Optional.of("not-a-backend"));
        LlamaCppServerManager local = mock(LlamaCppServerManager.class);
        var provider = provider(settings, local, mock(PiLlamaServerManager.class));

        assertThat(provider.getManager()).isSameAs(local);
    }

    private static LlmServerManagerProvider provider(AppSettingsStore settings,
                                                       LlamaCppServerManager local,
                                                       PiLlamaServerManager pi) {
        return new LlmServerManagerProvider(new LlmBackendSelector(settings, "local"), local, pi,
            mock(MlxServerManager.class));
    }
}
