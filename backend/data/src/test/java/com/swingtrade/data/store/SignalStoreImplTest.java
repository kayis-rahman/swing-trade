package com.swingtrade.data.store;

import com.swingtrade.data.entity.SignalEntity;
import com.swingtrade.data.repository.SignalRepository;
import com.swingtrade.domain.Signal;
import java.math.BigDecimal;
import java.time.LocalDate;
import java.util.List;
import java.util.Optional;
import java.util.Arrays;
import org.junit.jupiter.api.Test;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyList;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

class SignalStoreImplTest {
    private static final LocalDate DATE = LocalDate.of(2026, 1, 15);

    @Test
    void readsAndFiltersSignalsThroughRepository() {
        SignalRepository repository = mock(SignalRepository.class);
        SignalStoreImpl store = new SignalStoreImpl(repository);
        Signal signal = Signal.create("TCS", DATE, Signal.SignalType.BUY, bd(".8"), "entry");
        SignalEntity entity = mockEntity(signal, 12L, "PRICE_ACTION", 3);
        when(repository.findAll()).thenReturn(List.of(entity));
        when(repository.findBySymbolOrderByDateDesc(any(), any())).thenReturn(List.of(entity));
        when(repository.findBySymbolAndDate("TCS", DATE)).thenReturn(List.of(entity));
        when(repository.findStrategiesBySymbolAndDate("TCS", DATE)).thenReturn(List.of("PRICE_ACTION"));
        when(repository.findByDateRangeAndSignalType(any(), any(), any(), any())).thenReturn(List.of(entity));
        when(repository.findBySignalType(any(), any())).thenReturn(List.of(entity));
        when(repository.findUnprocessedBuySignalsSince(any())).thenReturn(List.of(entity));
        when(repository.findLatestBySymbol(any(), any())).thenReturn(List.of(entity));
        when(repository.findLatestBySymbolAndStrategy(any(), any(), any())).thenReturn(List.of(entity));
        when(repository.findLatestSignalPerSymbol()).thenReturn(List.of(entity));
        when(repository.findAllDistinctSymbols()).thenReturn(List.of("TCS"));
        when(repository.countBySymbolAndDate("TCS", DATE)).thenReturn(2L);
        when(repository.findBuySignalsSince(any(), any())).thenReturn(List.of(entity));
        when(repository.findByDateRange(any(), any(), any())).thenReturn(List.of(entity));
        when(repository.findByMinConfidence(any(), any())).thenReturn(List.of(entity));
        when(repository.findStrategyById(12L)).thenReturn(Optional.of("PRICE_ACTION"));
        when(repository.findAllById(anyList())).thenReturn(List.of(entity));

        assertThat(store.findAll()).containsExactly(signal);
        assertThat(store.findBySymbol("TCS")).containsExactly(signal);
        assertThat(store.findBySymbolOrderByDateDesc("TCS")).containsExactly(signal);
        assertThat(store.findBySymbolAndDate("TCS", DATE)).containsExactly(signal);
        assertThat(store.findStrategiesBySymbolAndDate("TCS", DATE)).containsExactly("PRICE_ACTION");
        assertThat(store.findByType(Signal.SignalType.BUY)).containsExactly(signal);
        verify(repository).findBySignalType(org.mockito.ArgumentMatchers.eq("BUY"), any());
        assertThat(store.findUnprocessed()).containsExactly(signal);
        assertThat(store.findLatestBySymbol("TCS")).contains(signal);
        assertThat(store.findLatestBySymbolAndStrategy("TCS", "PRICE_ACTION")).contains(signal);
        assertThat(store.findLatestSignalPerSymbol()).containsExactly(signal);
        assertThat(store.findAllDistinctSymbols()).containsExactly("TCS");
        assertThat(store.countBySymbolAndDate("TCS", DATE)).isEqualTo(2L);
        assertThat(store.findBuySignalsSince(DATE)).containsExactly(signal);
        assertThat(store.findByDateRange(DATE, DATE)).containsExactly(signal);
        assertThat(store.findByDateRangeAndType(DATE, DATE, Signal.SignalType.BUY)).containsExactly(signal);
        assertThat(store.findByMinConfidence(.7)).containsExactly(signal);
        assertThat(store.findStrategyById(12L)).contains("PRICE_ACTION");
        assertThat(store.findStrategyMetadataByIds(Arrays.asList(12L, null, 12L))).containsEntry(
                12L, new com.swingtrade.domain.SignalStrategyMetadata("PRICE_ACTION", 3));
    }

    @Test
    void savesVariantsAndMarksExistingSignalProcessed() {
        SignalRepository repository = mock(SignalRepository.class);
        SignalStoreImpl store = new SignalStoreImpl(repository);
        Signal signal = Signal.create("TCS", DATE, Signal.SignalType.BUY, bd(".8"), "entry");
        SignalEntity persisted = mockEntity(signal, 12L, "PRICE_ACTION", 3);
        when(repository.save(any(SignalEntity.class))).thenReturn(persisted);
        when(repository.findById(12L)).thenReturn(Optional.of(persisted));

        assertThat(store.save(signal)).isEqualTo(signal);
        assertThat(store.save(signal, "WARN")).isEqualTo(signal);
        assertThat(store.save(signal, "WARN", "PRICE_ACTION")).isEqualTo(signal);
        assertThat(store.save(signal, "WARN", "PRICE_ACTION", 4)).isEqualTo(signal);
        store.markProcessed(12L);
        verify(persisted).setProcessed(true);
        verify(repository).save(persisted);
    }

    @Test
    void handlesEmptyMetadataAndMissingStrategyIds() {
        SignalRepository repository = mock(SignalRepository.class);
        SignalStoreImpl store = new SignalStoreImpl(repository);
        assertThat(store.findStrategyMetadataByIds(null)).isEmpty();
        assertThat(store.findStrategyMetadataByIds(List.of())).isEmpty();
        when(repository.findStrategyById(99L)).thenReturn(Optional.empty());
        assertThat(store.findStrategyById(null)).isEmpty();
        assertThat(store.findStrategyById(99L)).isEmpty();
        when(repository.findById(99L)).thenReturn(Optional.empty());
        store.markProcessed(99L);
        verify(repository).findById(99L);
    }

    @Test
    void delegatesSignalDeletes() {
        SignalRepository repository = mock(SignalRepository.class);
        SignalStoreImpl store = new SignalStoreImpl(repository);
        when(repository.deleteBySymbolAndDate("TCS", DATE)).thenReturn(1);
        when(repository.deleteBySymbolAndDateAndStrategy("TCS", DATE, "PRICE_ACTION")).thenReturn(1);
        when(repository.deleteByDate(DATE)).thenReturn(2);

        assertThat(store.deleteBySymbolAndDate("TCS", DATE)).isEqualTo(1);
        assertThat(store.deleteBySymbolAndDateAndStrategy("TCS", DATE, "PRICE_ACTION")).isEqualTo(1);
        assertThat(store.deleteByDate(DATE)).isEqualTo(2);
        store.deleteAllSignals();
        verify(repository).deleteAllSignals();
    }

    private static SignalEntity mockEntity(Signal signal, long id, String strategy, int version) {
        SignalEntity entity = mock(SignalEntity.class);
        when(entity.toDomain()).thenReturn(signal);
        when(entity.getId()).thenReturn(id);
        when(entity.getStrategy()).thenReturn(strategy);
        when(entity.getStrategyVersion()).thenReturn(version);
        return entity;
    }

    private static BigDecimal bd(String value) {
        return new BigDecimal(value);
    }
}
