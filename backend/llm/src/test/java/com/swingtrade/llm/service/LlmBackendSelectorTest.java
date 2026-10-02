package com.swingtrade.llm.service;

import com.swingtrade.domain.store.AppSettingsStore;
import org.junit.jupiter.api.Test;

import java.util.Optional;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.when;

class LlmBackendSelectorTest {
    @Test
    void resolvesPersistedBackendBeforeSpringDefault() {
        AppSettingsStore settings = mock(AppSettingsStore.class);
        when(settings.get("llm.backend")).thenReturn(Optional.of("openai"));

        assertThat(new LlmBackendSelector(settings, "local").resolve())
            .isEqualTo(LlmBackendSelector.Backend.OPENAI);
    }

    @Test
    void resolvesDefaultAndAliasesSafely() {
        AppSettingsStore settings = mock(AppSettingsStore.class);
        when(settings.get("llm.backend")).thenReturn(Optional.empty());

        assertThat(new LlmBackendSelector(settings, "pi_ssh").resolve())
            .isEqualTo(LlmBackendSelector.Backend.PI_SSH);
        assertThat(LlmBackendSelector.fromKey("gpuhub")).isEqualTo(LlmBackendSelector.Backend.OPENAI);
        assertThat(LlmBackendSelector.fromKey("unknown")).isEqualTo(LlmBackendSelector.Backend.LOCAL);
    }

    @Test
    void resolvesLayaBackend() {
        AppSettingsStore settings = mock(AppSettingsStore.class);
        when(settings.get("llm.backend")).thenReturn(Optional.of("laya"));

        assertThat(new LlmBackendSelector(settings, "local").resolve())
            .isEqualTo(LlmBackendSelector.Backend.LAYA);
        assertThat(LlmBackendSelector.fromKey("laya")).isEqualTo(LlmBackendSelector.Backend.LAYA);
    }
}
