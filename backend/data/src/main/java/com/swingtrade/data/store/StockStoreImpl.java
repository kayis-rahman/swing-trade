package com.swingtrade.data.store;

import com.swingtrade.data.entity.StockEntity;
import com.swingtrade.data.repository.StockRepository;
import com.swingtrade.domain.Stock;
import com.swingtrade.domain.store.StockStore;
import org.springframework.stereotype.Service;

import java.time.LocalDate;
import java.util.List;
import java.util.Optional;

@Service
public class StockStoreImpl implements StockStore {

    private final StockRepository repository;

    public StockStoreImpl(StockRepository repository) {
        this.repository = repository;
    }

    @Override
    public List<Stock> findAllActive() {
        return repository.findAllByOrderBySymbol().stream()
            .map(StockEntity::toDomain)
            .toList();
    }

    @Override
    public Optional<Stock> findBySymbol(String symbol) {
        return repository.findBySymbol(symbol).map(StockEntity::toDomain);
    }

    @Override
    public void save(Stock stock) {
        repository.save(StockEntity.fromDomain(stock));
    }

    @Override
    public boolean existsBySymbol(String symbol) {
        return repository.existsBySymbol(symbol);
    }

    @Override
    public List<Stock> findBySector(String sector) {
        return repository.findBySector(sector).stream()
            .map(StockEntity::toDomain)
            .toList();
    }

    @Override
    public List<Stock> findByAddedOnBefore(LocalDate addedOn) {
        return repository.findByAddedOnBefore(addedOn).stream()
            .map(StockEntity::toDomain)
            .toList();
    }

    @Override
    public List<Stock> findAllByOrderBySymbol() {
        return repository.findAllByOrderBySymbol().stream()
            .map(StockEntity::toDomain)
            .toList();
    }

    @Override
    public List<String> findAllDistinctSymbols() {
        return repository.findAllDistinctSymbols();
    }
}