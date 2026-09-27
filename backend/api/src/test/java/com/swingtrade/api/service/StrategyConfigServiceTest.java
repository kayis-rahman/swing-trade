package com.swingtrade.api.service;

import com.swingtrade.api.dto.StrategyConfigRequest;
import com.swingtrade.api.dto.StrategyModeRequest;
import com.swingtrade.data.entity.StrategyConfigEntity;
import com.swingtrade.data.repository.StrategyConfigRepository;
import com.swingtrade.domain.StrategyConfig;
import com.swingtrade.domain.service.VariantTradingService;
import com.swingtrade.domain.store.StrategyConfigStore;
import jakarta.persistence.EntityManager;
import jakarta.persistence.Query;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.math.BigDecimal;
import java.time.LocalDateTime;
import java.util.List;
import java.util.Map;
import java.util.Optional;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

class StrategyConfigServiceTest {
    private StrategyConfigStore store;
    private StrategyConfigRepository repository;
    private EntityManager entityManager;
    private Query clearCurrentQuery;
    private StrategyConfigService service;

    @BeforeEach
    void setUp() {
        store = mock(StrategyConfigStore.class);
        repository = mock(StrategyConfigRepository.class);
        entityManager = mock(EntityManager.class);
        clearCurrentQuery = mock(Query.class);
        when(entityManager.createQuery(any(String.class))).thenReturn(clearCurrentQuery);
        when(clearCurrentQuery.setParameter(any(String.class), any())).thenReturn(clearCurrentQuery);
        when(clearCurrentQuery.executeUpdate()).thenReturn(1);
        when(repository.findAll()).thenReturn(List.of());
        service = new StrategyConfigService(store, repository, entityManager, mock(VariantTradingService.class));
    }

    @Test
    void createsFirstVersionWithHashAndCurrentFlag() {
        StrategyConfig saved = config("BREAKOUT", 1, StrategyConfig.Mode.SHADOW, true);
        when(store.save(any(StrategyConfig.class))).thenReturn(saved);

        var response = service.create(request("BREAKOUT", null, StrategyConfig.Mode.SHADOW));

        assertThat(response.variantId()).isEqualTo("BREAKOUT");
        assertThat(response.paramsHash()).hasSize(64);
        verify(clearCurrentQuery).setParameter("variantId", "BREAKOUT");
        verify(store).save(any(StrategyConfig.class));
    }

    @Test
    void modeChangeCreatesNextVersionAndRetiresPriorCurrentVersion() {
        StrategyConfig current = config("BREAKOUT", 2, StrategyConfig.Mode.SHADOW, true);
        StrategyConfig saved = config("BREAKOUT", 3, StrategyConfig.Mode.CHAMPION, true);
        when(store.findCurrentByVariantId("BREAKOUT")).thenReturn(Optional.of(current));
        when(store.save(any(StrategyConfig.class))).thenReturn(saved);

        var response = service.changeMode("BREAKOUT", new StrategyModeRequest(StrategyConfig.Mode.CHAMPION, null));

        assertThat(response.version()).isEqualTo(3);
        assertThat(response.mode()).isEqualTo(StrategyConfig.Mode.CHAMPION);
        verify(clearCurrentQuery).executeUpdate();
    }

    @Test
    void deleteRetiresVariantAsOffInANewVersion() {
        StrategyConfig current = config("AUDIT", 2, StrategyConfig.Mode.OFF, true);
        StrategyConfig saved = config("AUDIT", 3, StrategyConfig.Mode.OFF, true);
        when(store.findCurrentByVariantId("AUDIT")).thenReturn(Optional.of(current));
        when(store.save(any(StrategyConfig.class))).thenReturn(saved);

        var response = service.delete("AUDIT");

        assertThat(response.version()).isEqualTo(3);
        assertThat(response.mode()).isEqualTo(StrategyConfig.Mode.OFF);
        verify(clearCurrentQuery).executeUpdate();
    }

    @Test
    void rejectsSecondChampion() {
        StrategyConfig champion = config("OTHER", 1, StrategyConfig.Mode.CHAMPION, true);
        when(repository.findAll()).thenReturn(List.of(StrategyConfigEntity.fromDomain(champion)));

        assertThatThrownBy(() -> service.create(request("BREAKOUT", null, StrategyConfig.Mode.CHAMPION)))
            .isInstanceOf(IllegalArgumentException.class)
            .hasMessageContaining("one current CHAMPION");
    }

    @Test
    void rejectsPromotingAnotherCurrentVariantWhenChampionAlreadyExists() {
        StrategyConfig champion = config("BREAKOUT", 1, StrategyConfig.Mode.CHAMPION, true);
        StrategyConfig shadow = config("PULLBACK", 1, StrategyConfig.Mode.SHADOW, true);
        when(repository.findAll()).thenReturn(List.of(
            StrategyConfigEntity.fromDomain(champion), StrategyConfigEntity.fromDomain(shadow)));
        when(store.findCurrentByVariantId("PULLBACK")).thenReturn(Optional.of(shadow));

        assertThatThrownBy(() -> service.changeMode("PULLBACK",
            new StrategyModeRequest(StrategyConfig.Mode.CHAMPION, null)))
            .isInstanceOf(IllegalArgumentException.class)
            .hasMessageContaining("one current CHAMPION");
    }

    @Test
    void rejectsThirteenthActiveVariant() {
        List<StrategyConfigEntity> active = java.util.stream.IntStream.range(0, 12)
            .mapToObj(i -> StrategyConfigEntity.fromDomain(
                config("V" + i, 1, StrategyConfig.Mode.SHADOW, true)))
            .toList();
        when(repository.findAll()).thenReturn(active);

        assertThatThrownBy(() -> service.create(request("NEW", null, StrategyConfig.Mode.SHADOW)))
            .isInstanceOf(IllegalArgumentException.class)
            .hasMessageContaining("At most 12");
    }

    private static StrategyConfigRequest request(String variantId, Integer version, StrategyConfig.Mode mode) {
        return new StrategyConfigRequest(variantId, version, "BREAKOUT", Map.of("rsiMin", 50),
            Map.of(), mode, BigDecimal.valueOf(500_000), null);
    }

    private static StrategyConfig config(String variantId, int version, StrategyConfig.Mode mode, boolean current) {
        return StrategyConfig.create(variantId, version, "BREAKOUT", Map.of("rsiMin", 50),
            Map.of(), mode, BigDecimal.valueOf(500_000), current, null, LocalDateTime.now());
    }
}
