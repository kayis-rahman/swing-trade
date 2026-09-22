package com.swingtrade.data.entity;

import jakarta.persistence.Column;
import jakarta.persistence.Entity;
import jakarta.persistence.GeneratedValue;
import jakarta.persistence.GenerationType;
import jakarta.persistence.Id;
import jakarta.persistence.Index;
import jakarta.persistence.Table;
import jakarta.persistence.Version;

import java.time.LocalDate;
import java.time.OffsetDateTime;

/**
 * One row per sentiment classification call (Laya or Qwen), used to compute
 * the Laya/Qwen agreement rate during Laya's shadow-mode rollout.
 *
 * <p>This is a distinct table from {@link SentimentAccuracyEntity}, which is
 * a post-hoc accuracy table (ground truth vs. LLM prediction) unrelated to
 * Laya. Do not conflate the two.
 */
@Entity
@Table(name = "sentiment_classification_log", indexes = {
    @Index(name = "idx_sentiment_classification_log_symbol_date", columnList = "symbol, analysis_date"),
    @Index(name = "idx_sentiment_classification_log_model", columnList = "model_used"),
    @Index(name = "idx_sentiment_classification_log_created", columnList = "created_at")
})
public class SentimentClassificationLogEntity {

    /** Which model produced this row's sentiment. */
    public enum ModelUsed { LAYA, QWEN }

    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Long id;

    @Version
    private Integer version = 0;

    @Column(nullable = false, length = 20)
    private String symbol;

    @Column(name = "analysis_date", nullable = false)
    private LocalDate analysisDate;

    @Column(name = "model_used", nullable = false, length = 10)
    private String modelUsed;

    @Column(nullable = false, length = 10)
    private String sentiment;

    @Column
    private Double confidence;

    @Column(name = "latency_ms")
    private Long latencyMs;

    @Column(name = "shadow_mode", nullable = false)
    private Boolean shadowMode = Boolean.TRUE;

    @Column(name = "created_at", updatable = false)
    private OffsetDateTime createdAt;

    public SentimentClassificationLogEntity() {
    }

    public SentimentClassificationLogEntity(String symbol, LocalDate analysisDate, ModelUsed modelUsed,
                                            String sentiment, Double confidence, Long latencyMs,
                                            boolean shadowMode) {
        this.symbol = symbol;
        this.analysisDate = analysisDate;
        this.modelUsed = modelUsed.name();
        this.sentiment = sentiment;
        this.confidence = confidence;
        this.latencyMs = latencyMs;
        this.shadowMode = shadowMode;
        this.createdAt = OffsetDateTime.now();
    }

    public Long getId() {
        return id;
    }

    public void setId(Long id) {
        this.id = id;
    }

    public String getSymbol() {
        return symbol;
    }

    public void setSymbol(String symbol) {
        this.symbol = symbol;
    }

    public LocalDate getAnalysisDate() {
        return analysisDate;
    }

    public void setAnalysisDate(LocalDate analysisDate) {
        this.analysisDate = analysisDate;
    }

    public String getModelUsed() {
        return modelUsed;
    }

    public void setModelUsed(String modelUsed) {
        this.modelUsed = modelUsed;
    }

    public String getSentiment() {
        return sentiment;
    }

    public void setSentiment(String sentiment) {
        this.sentiment = sentiment;
    }

    public Double getConfidence() {
        return confidence;
    }

    public void setConfidence(Double confidence) {
        this.confidence = confidence;
    }

    public Long getLatencyMs() {
        return latencyMs;
    }

    public void setLatencyMs(Long latencyMs) {
        this.latencyMs = latencyMs;
    }

    public Boolean getShadowMode() {
        return shadowMode;
    }

    public void setShadowMode(Boolean shadowMode) {
        this.shadowMode = shadowMode;
    }

    public OffsetDateTime getCreatedAt() {
        return createdAt;
    }

    public void setCreatedAt(OffsetDateTime createdAt) {
        this.createdAt = createdAt;
    }
}
