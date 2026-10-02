package com.swingtrade.data.entity;

import com.swingtrade.domain.StrategyConfig;
import jakarta.persistence.Column;
import jakarta.persistence.Entity;
import jakarta.persistence.EnumType;
import jakarta.persistence.Enumerated;
import jakarta.persistence.GeneratedValue;
import jakarta.persistence.GenerationType;
import jakarta.persistence.Id;
import jakarta.persistence.Table;
import jakarta.persistence.Version;
import org.hibernate.annotations.JdbcTypeCode;
import org.hibernate.type.SqlTypes;

import java.math.BigDecimal;
import java.time.LocalDateTime;
import java.util.Map;

@Entity
@Table(name = "strategy_config")
public class StrategyConfigEntity {
    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Long id;

    @Version
    @Column(name = "lock_version", nullable = false)
    private Integer lockVersion = 0;

    @Column(name = "variant_id", nullable = false, length = 40)
    private String variantId;

    @Column(nullable = false)
    private int version;

    @Column(name = "strategy_type", nullable = false, length = 30)
    private String strategyType;

    @JdbcTypeCode(SqlTypes.JSON)
    @Column(nullable = false, columnDefinition = "jsonb")
    private Map<String, Object> params;

    @JdbcTypeCode(SqlTypes.JSON)
    @Column(nullable = false, columnDefinition = "jsonb")
    private Map<String, Object> overlays;

    @Column(name = "params_hash", nullable = false, length = 64)
    private String paramsHash;

    @Enumerated(EnumType.STRING)
    @Column(nullable = false, length = 16)
    private StrategyConfig.Mode mode;

    @Column(name = "paper_capital", nullable = false, precision = 15, scale = 2)
    private BigDecimal paperCapital;

    @Column(name = "is_current", nullable = false)
    private boolean current;

    @Column(columnDefinition = "TEXT")
    private String notes;

    @Column(name = "created_at", nullable = false)
    private LocalDateTime createdAt;

    protected StrategyConfigEntity() {}

    public static StrategyConfigEntity fromDomain(StrategyConfig config) {
        StrategyConfigEntity entity = new StrategyConfigEntity();
        entity.id = config.id();
        entity.variantId = config.variantId();
        entity.version = config.version();
        entity.strategyType = config.strategyType();
        entity.params = config.params();
        entity.overlays = config.overlays();
        entity.paramsHash = config.paramsHash();
        entity.mode = config.mode();
        entity.paperCapital = config.paperCapital();
        entity.current = config.current();
        entity.notes = config.notes();
        entity.createdAt = config.createdAt();
        return entity;
    }

    public StrategyConfig toDomain() {
        return new StrategyConfig(id, variantId, version, strategyType, params, overlays,
            paramsHash, mode, paperCapital, current, notes, createdAt);
    }

    public Long getId() { return id; }
    public String getVariantId() { return variantId; }
    public int getVersion() { return version; }
    public String getStrategyType() { return strategyType; }
    public Map<String, Object> getParams() { return params; }
    public Map<String, Object> getOverlays() { return overlays; }
    public String getParamsHash() { return paramsHash; }
    public StrategyConfig.Mode getMode() { return mode; }
    public BigDecimal getPaperCapital() { return paperCapital; }
    public boolean isCurrent() { return current; }
    public String getNotes() { return notes; }
    public LocalDateTime getCreatedAt() { return createdAt; }
}
