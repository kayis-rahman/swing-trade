package com.swingtrade.data.entity;

import com.swingtrade.domain.JobRun;
import com.swingtrade.domain.JobRunStage;
import com.swingtrade.domain.SentimentAccuracy;
import com.swingtrade.domain.Stock;
import java.math.BigDecimal;
import java.time.LocalDate;
import java.time.LocalDateTime;
import java.util.UUID;
import org.junit.jupiter.api.Test;

import static org.assertj.core.api.Assertions.assertThat;

class EntityPersistenceContractTest {
    private static final LocalDate DATE = LocalDate.of(2026, 1, 15);
    private static final LocalDateTime TIME = LocalDateTime.of(2026, 1, 15, 10, 30);

    @Test
    void stockRoundTripsDomainFields() {
        Stock stock = new Stock("TCS", Stock.Exchange.NSE, "TCS Limited", Stock.Sector.IT,
                "Software", 1_000_000L, new BigDecimal("28.50"), "INE467B01029", 1, DATE);

        StockEntity entity = StockEntity.fromDomain(stock);

        assertThat(entity.toDomain()).isEqualTo(stock);
        entity.setId(4L);
        entity.setCreatedAt(TIME);
        entity.setUpdatedAt(TIME.plusMinutes(1));
        assertThat(entity.getId()).isEqualTo(4L);
        assertThat(entity.getCreatedAt()).isEqualTo(TIME);
        assertThat(entity.getUpdatedAt()).isEqualTo(TIME.plusMinutes(1));
    }

    @Test
    void sentimentAccuracyRoundTripsReturnsAndProvenance() {
        SentimentAccuracy accuracy = new SentimentAccuracy(7L, "TCS", DATE, "POSITIVE", .8f, .4f,
                new BigDecimal(".01"), new BigDecimal(".03"), new BigDecimal(".08"),
                new BigDecimal(".005"), new BigDecimal(".02"), new BigDecimal(".06"),
                "EXCESS_RETURN", "UP", true, new BigDecimal(".08"), "BULL", "hash", "model",
                "NEWS", TIME);

        SentimentAccuracyEntity entity = SentimentAccuracyEntity.fromDomain(accuracy);

        assertThat(entity.toDomain()).isEqualTo(accuracy);
        entity.setId(7L);
        entity.setCompositeScore(80);
        entity.setCompositeSignal("BUY");
        entity.setCompositeId(9L);
        entity.setCreatedAt(TIME);
        assertThat(entity.getId()).isEqualTo(7L);
        assertThat(entity.getCompositeScore()).isEqualTo(80);
        assertThat(entity.getCompositeSignal()).isEqualTo("BUY");
        assertThat(entity.getCompositeId()).isEqualTo(9L);
        assertThat(entity.getCreatedAt()).isEqualTo(TIME);
    }

    @Test
    void jobRunAndStageRoundTripDomainFields() {
        UUID runId = UUID.randomUUID();
        JobRun run = new JobRun(runId, JobRun.TriggerType.MANUAL, JobRun.Status.FAILED,
                TIME, TIME.plusMinutes(5), 10, 8, 2, "two failures");
        JobRunEntity runEntity = JobRunEntity.fromDomain(run);
        assertThat(runEntity.toDomain()).isEqualTo(run);
        runEntity.setId(3L);
        runEntity.setCandidateScanRunId(UUID.randomUUID());
        assertThat(runEntity.getId()).isEqualTo(3L);
        assertThat(runEntity.getCandidateScanRunId()).isNotNull();

        JobRunStage stage = new JobRunStage(runId, "TCS", JobRunStage.StageName.SIGNAL,
                JobRunStage.Status.ERROR, TIME, TIME.plusSeconds(2), 2_000L, "failed",
                "details", "summary");
        JobRunStageEntity stageEntity = JobRunStageEntity.fromDomain(stage);
        assertThat(stageEntity.toDomain()).isEqualTo(stage);
        stageEntity.setId(5L);
        assertThat(stageEntity.getId()).isEqualTo(5L);
    }

    @Test
    void simpleEntitiesExposeAllPersistenceFields() {
        AppSettingEntity setting = new AppSettingEntity("trading.initial-capital", "500000");
        setting.setId(1L);
        setting.setUpdatedAt(TIME);
        assertThat(setting.getId()).isEqualTo(1L);
        assertThat(setting.getKey()).isEqualTo("trading.initial-capital");
        assertThat(setting.getValue()).isEqualTo("500000");
        assertThat(setting.getUpdatedAt()).isEqualTo(TIME);

        NewsItemEntity news = new NewsItemEntity();
        news.setId(2L); news.setSymbol("TCS"); news.setHeadline("Results"); news.setSource("NSE");
        news.setPublishedAt(TIME); news.setUrl("https://example.test/news");
        news.setRawContent("content"); news.setCreatedAt(TIME);
        assertThat(news.getId()).isEqualTo(2L); assertThat(news.getSymbol()).isEqualTo("TCS");
        assertThat(news.getHeadline()).isEqualTo("Results"); assertThat(news.getSource()).isEqualTo("NSE");
        assertThat(news.getPublishedAt()).isEqualTo(TIME); assertThat(news.getUrl()).contains("example");
        assertThat(news.getRawContent()).isEqualTo("content"); assertThat(news.getCreatedAt()).isEqualTo(TIME);

        PdfExtractionEntity pdf = new PdfExtractionEntity();
        pdf.setId(3L); pdf.setSymbol("TCS"); pdf.setDocumentType("RESULTS");
        pdf.setExtractedJson("{}"); pdf.setSourceUrl("https://example.test/pdf");
        pdf.setExtractionDate(DATE); pdf.setModelUsed("test-model"); pdf.setCreatedAt(TIME);
        assertThat(pdf.getId()).isEqualTo(3L); assertThat(pdf.getSymbol()).isEqualTo("TCS");
        assertThat(pdf.getDocumentType()).isEqualTo("RESULTS"); assertThat(pdf.getExtractedJson()).isEqualTo("{}");
        assertThat(pdf.getSourceUrl()).contains("example"); assertThat(pdf.getExtractionDate()).isEqualTo(DATE);
        assertThat(pdf.getModelUsed()).isEqualTo("test-model"); assertThat(pdf.getCreatedAt()).isEqualTo(TIME);

        NseHolidayEntity holiday = new NseHolidayEntity(DATE, "Festival", "FULL");
        holiday.setId(4L); holiday.setCreatedAt(TIME);
        assertThat(holiday.getId()).isEqualTo(4L); assertThat(holiday.getHolidayDate()).isEqualTo(DATE);
        assertThat(holiday.getOccasion()).isEqualTo("Festival"); assertThat(holiday.getHolidayType()).isEqualTo("FULL");
        assertThat(holiday.getCreatedAt()).isEqualTo(TIME);

        new ReconciliationAuditEntity("TCS", DATE, DATE.plusDays(1), "COMPLETED", 1, 2, 0);
    }

    @Test
    void scanRunStoresProgressAndOrchestrationState() {
        CandidateScanRunEntity run = new CandidateScanRunEntity();
        UUID runId = UUID.randomUUID();
        UUID jobId = UUID.randomUUID();
        run.setRunId(runId); run.setStatus("COMPLETED"); run.setScanScope("WATCHLIST"); run.setTotalSymbols(10);
        run.setCompletedSymbols(9); run.setFailedSymbols(1); run.setQualifiedSymbols(4);
        run.setStartedAt(TIME); run.setCompletedAt(TIME.plusMinutes(3)); run.setErrorMessage("one failed");
        run.setOrchestrationStatus("COMPLETED"); run.setOrchestrationJobRunId(jobId);
        run.setOrchestrationError(null);
        assertThat(run.getRunId()).isEqualTo(runId); assertThat(run.getStatus()).isEqualTo("COMPLETED");
        assertThat(run.getScanScope()).isEqualTo("WATCHLIST");
        assertThat(run.getTotalSymbols()).isEqualTo(10); assertThat(run.getCompletedSymbols()).isEqualTo(9);
        assertThat(run.getFailedSymbols()).isEqualTo(1); assertThat(run.getQualifiedSymbols()).isEqualTo(4);
        assertThat(run.getStartedAt()).isEqualTo(TIME); assertThat(run.getCompletedAt()).isEqualTo(TIME.plusMinutes(3));
        assertThat(run.getErrorMessage()).isEqualTo("one failed");
        assertThat(run.getOrchestrationStatus()).isEqualTo("COMPLETED");
        assertThat(run.getOrchestrationJobRunId()).isEqualTo(jobId); assertThat(run.getOrchestrationError()).isNull();
    }

    @Test
    void environmentSettingsPreferNonBlankSystemProperty() {
        String key = "entity.contract.test." + UUID.randomUUID();
        String previous = System.getProperty(key);
        try {
            System.setProperty(key, "configured");
            assertThat(AppSettingEntity.fromEnv(key, "fallback")).isEqualTo("configured");
        } finally {
            if (previous == null) System.clearProperty(key);
            else System.setProperty(key, previous);
        }
    }
}
