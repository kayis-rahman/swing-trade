package com.swingtrade.api.controller;

import com.swingtrade.broker.risk.KillSwitchService;
import com.swingtrade.data.repository.FyersSymbolRepository;
import com.swingtrade.data.service.DataIngestionService;
import org.junit.jupiter.api.Test;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.setup.MockMvcBuilders;
import org.springframework.web.reactive.function.client.WebClient;

import java.time.LocalDate;

import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;
import static org.springframework.http.MediaType.APPLICATION_JSON;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

class AdminControllerDataQualityTest {

    private final DataIngestionService ingestionService = mock(DataIngestionService.class);
    private final KillSwitchService killSwitchService = mock(KillSwitchService.class);
    private final MockMvc mockMvc = MockMvcBuilders.standaloneSetup(new AdminController(
            killSwitchService, ingestionService, WebClient.builder(),
            mock(FyersSymbolRepository.class))).build();

    @Test
    void validatesAndNormalizesSymbolBeforeDelegating() throws Exception {
        DataIngestionService.DataQualityReport report = new DataIngestionService.DataQualityReport();
        when(ingestionService.validateDataQuality(eq("TCS"), eq(LocalDate.of(2026, 1, 1)),
                eq(LocalDate.of(2026, 1, 31)))).thenReturn(report);

        mockMvc.perform(post("/api/admin/data/validate")
                        .contentType(APPLICATION_JSON)
                        .content("{\"symbol\":\" tcs \",\"fromDate\":\"2026-01-01\",\"toDate\":\"2026-01-31\"}"))
                .andExpect(status().isOk());

        verify(ingestionService).validateDataQuality("TCS", LocalDate.of(2026, 1, 1),
                LocalDate.of(2026, 1, 31));
    }

    @Test
    void rejectsMissingOrReversedWindow() throws Exception {
        mockMvc.perform(post("/api/admin/data/validate")
                        .contentType(APPLICATION_JSON)
                        .content("{\"symbol\":\"TCS\",\"fromDate\":\"2026-02-01\",\"toDate\":\"2026-01-31\"}"))
                .andExpect(status().isBadRequest());
    }

    @Test
    void allowsDisablingKillSwitchWhenNullableStateIsCleared() throws Exception {
        when(killSwitchService.isActive()).thenReturn(false);

        mockMvc.perform(post("/api/admin/kill-switch/toggle")
                        .param("enable", "false"))
                .andExpect(status().isOk());

        verify(killSwitchService).toggle(false);
    }
}
