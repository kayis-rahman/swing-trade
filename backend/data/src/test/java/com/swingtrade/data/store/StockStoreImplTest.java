package com.swingtrade.data.store;

import com.swingtrade.data.entity.StockEntity;
import com.swingtrade.data.repository.StockRepository;
import com.swingtrade.domain.Stock;
import org.junit.jupiter.api.Test;

import java.time.LocalDate;
import java.util.List;
import java.util.Optional;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

class StockStoreImplTest {
    private static final Stock STOCK = new Stock("TCS", Stock.Exchange.NSE, "TCS", Stock.Sector.IT,
        "Software", 1_000L, null, "INE467B01029", 1, LocalDate.of(2020, 1, 1));

    @Test
    void mapsQueriesAndDelegatesSave() {
        StockRepository repository = mock(StockRepository.class);
        StockEntity entity = StockEntity.fromDomain(STOCK);
        when(repository.findAllByOrderBySymbol()).thenReturn(List.of(entity));
        when(repository.findBySymbol("TCS")).thenReturn(Optional.of(entity));
        when(repository.findBySector("IT")).thenReturn(List.of(entity));
        when(repository.findByAddedOnBefore(LocalDate.of(2021, 1, 1))).thenReturn(List.of(entity));
        when(repository.findAllDistinctSymbols()).thenReturn(List.of("TCS"));
        StockStoreImpl store = new StockStoreImpl(repository);

        assertThat(store.findAllActive()).extracting(Stock::symbol).containsExactly("TCS");
        assertThat(store.findBySymbol("TCS")).get().extracting(Stock::symbol).isEqualTo("TCS");
        assertThat(store.findBySector("IT")).hasSize(1);
        assertThat(store.findByAddedOnBefore(LocalDate.of(2021, 1, 1))).hasSize(1);
        assertThat(store.findAllByOrderBySymbol()).hasSize(1);
        assertThat(store.findAllDistinctSymbols()).containsExactly("TCS");
        assertThat(store.existsBySymbol("TCS")).isFalse();
        store.save(STOCK);
        verify(repository).save(any(StockEntity.class));
    }
}
