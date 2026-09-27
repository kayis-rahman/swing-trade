package com.swingtrade.api.service;

import com.swingtrade.api.dto.StrategyConfigRequest;
import com.swingtrade.api.dto.StrategyConfigResponse;
import com.swingtrade.api.dto.StrategyModeRequest;
import com.swingtrade.data.entity.StrategyConfigEntity;
import com.swingtrade.data.repository.StrategyConfigRepository;
import com.swingtrade.domain.StrategyConfig;
import com.swingtrade.domain.service.VariantTradingService;
import com.swingtrade.domain.store.StrategyConfigStore;
import jakarta.persistence.EntityManager;
import jakarta.transaction.Transactional;
import org.springframework.stereotype.Service;

import java.time.LocalDateTime;
import java.util.Comparator;
import java.util.List;

/** API orchestration for append-only, versioned strategy configurations. */
@Service
public class StrategyConfigService {
    public static final int MAX_ACTIVE_VARIANTS = 12;

    private final StrategyConfigStore store;
    private final StrategyConfigRepository repository;
    private final EntityManager entityManager;
    private final VariantTradingService variantTradingService;

    public StrategyConfigService(StrategyConfigStore store,
                                 StrategyConfigRepository repository,
                                 EntityManager entityManager,
                                 VariantTradingService variantTradingService) {
        this.store = store;
        this.repository = repository;
        this.entityManager = entityManager;
        this.variantTradingService = variantTradingService;
    }

    /**
     * Auto-creates the variant's own paper-trading portfolio the moment it becomes
     * SHADOW or CHAMPION (task constraint 1). Idempotent and isolated: a failure here
     * must not block the strategy-config write itself, since the portfolio can still
     * be created lazily on the next job run if this best-effort call fails.
     */
    private void ensurePaperPortfolioIfLive(StrategyConfig config) {
        if (variantTradingService == null || config == null) return;
        if (config.mode() == StrategyConfig.Mode.SHADOW || config.mode() == StrategyConfig.Mode.CHAMPION) {
            try {
                variantTradingService.ensurePortfolio(config.variantId(), config.paperCapital());
            } catch (RuntimeException e) {
                // Best-effort: the job orchestrator's PAPER_TRADE stage also calls
                // ensurePortfolio() on every run, so a transient failure here is not fatal.
            }
        }
    }

    public List<StrategyConfigResponse> list(String variantId, StrategyConfig.Mode mode) {
        return (variantId == null || variantId.isBlank()
            ? repository.findAll().stream().map(StrategyConfigEntity::toDomain).toList()
            : store.findByVariantId(variantId))
            .stream()
            .filter(config -> mode == null || config.mode() == mode)
            .sorted(Comparator.comparing(StrategyConfig::variantId).thenComparing(StrategyConfig::version).reversed())
            .map(StrategyConfigResponse::from)
            .toList();
    }

    public StrategyConfigResponse current(String variantId) {
        return store.findCurrentByVariantId(normalizeVariantId(variantId))
            .map(StrategyConfigResponse::from)
            .orElseThrow(() -> new IllegalArgumentException("Strategy variant not found: " + variantId));
    }

    public List<StrategyConfigResponse> versions(String variantId) {
        requireVariant(variantId);
        return store.findByVariantId(variantId).stream().map(StrategyConfigResponse::from).toList();
    }

    public StrategyConfigResponse version(String variantId, int version) {
        requireVariant(variantId);
        return store.findByVariantIdAndVersion(variantId, version)
            .map(StrategyConfigResponse::from)
            .orElseThrow(() -> new IllegalArgumentException(
                "Strategy configuration version not found: " + variantId + "/" + version));
    }

    @Transactional
    public StrategyConfigResponse create(StrategyConfigRequest request) {
        String variantId = normalizeVariantId(request.variantId());
        StrategyConfig current = store.findCurrentByVariantId(variantId).orElse(null);
        int version = request.version() == null
            ? (current == null ? 1 : current.version() + 1)
            : request.version();
        if (current != null && version <= current.version()) {
            throw new IllegalArgumentException("version must be greater than the current version");
        }
        enforceCapacity(variantId, request.mode());
        clearCurrent(variantId);
        StrategyConfig config = StrategyConfig.create(variantId, version, request.strategyType().trim(),
            request.params(), request.normalizedOverlays(), request.mode(), request.paperCapital(), true,
            request.notes(), LocalDateTime.now());
        StrategyConfig saved = store.save(config);
        ensurePaperPortfolioIfLive(saved);
        return StrategyConfigResponse.from(saved);
    }

    @Transactional
    public StrategyConfigResponse changeMode(String variantId, StrategyModeRequest request) {
        String normalized = normalizeVariantId(variantId);
        StrategyConfig current = store.findCurrentByVariantId(normalized)
            .orElseThrow(() -> new IllegalArgumentException("Strategy variant not found: " + variantId));
        if (current.mode() == request.mode() && (request.notes() == null || request.notes().equals(current.notes()))) {
            return StrategyConfigResponse.from(current);
        }
        enforceCapacity(normalized, request.mode());
        clearCurrent(normalized);
        StrategyConfig next = StrategyConfig.create(normalized, current.version() + 1, current.strategyType(),
            current.params(), current.overlays(), request.mode(), current.paperCapital(), true,
            request.notes() == null ? current.notes() : request.notes(), LocalDateTime.now());
        StrategyConfig saved = store.save(next);
        ensurePaperPortfolioIfLive(saved);
        return StrategyConfigResponse.from(saved);
    }

    @Transactional
    public StrategyConfigResponse delete(String variantId) {
        return changeMode(variantId, new StrategyModeRequest(StrategyConfig.Mode.OFF, "Disabled via API"));
    }

    private void enforceCapacity(String variantId, StrategyConfig.Mode mode) {
        if (mode == StrategyConfig.Mode.CHAMPION && repository.findAll().stream()
            .map(StrategyConfigEntity::toDomain)
            .anyMatch(config -> config.current() && config.mode() == StrategyConfig.Mode.CHAMPION
                && !config.variantId().equals(variantId))) {
            throw new IllegalArgumentException("Only one current CHAMPION strategy is allowed");
        }
        if (mode != StrategyConfig.Mode.OFF && repository.findAll().stream()
            .map(StrategyConfigEntity::toDomain)
            .filter(config -> config.current() && config.mode() != StrategyConfig.Mode.OFF)
            .map(StrategyConfig::variantId)
            .distinct().count() >= MAX_ACTIVE_VARIANTS
            && store.findCurrentByVariantId(variantId).map(c -> c.mode() == StrategyConfig.Mode.OFF).orElse(true)) {
            throw new IllegalArgumentException("At most " + MAX_ACTIVE_VARIANTS + " active strategy variants are allowed");
        }
    }

    private void clearCurrent(String variantId) {
        entityManager.createQuery("update StrategyConfigEntity c set c.current = false where c.variantId = :variantId")
            .setParameter("variantId", variantId).executeUpdate();
        entityManager.clear();
    }

    private static String normalizeVariantId(String value) {
        if (value == null || value.isBlank()) throw new IllegalArgumentException("variantId is required");
        return value.trim();
    }

    private static void requireVariant(String value) { normalizeVariantId(value); }
}
