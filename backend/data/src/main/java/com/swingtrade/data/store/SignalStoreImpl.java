package com.swingtrade.data.store;

import com.swingtrade.data.entity.SignalEntity;
import com.swingtrade.data.repository.SignalRepository;
import com.swingtrade.domain.Signal;
import com.swingtrade.domain.store.SignalStore;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.math.BigDecimal;
import java.time.LocalDate;
import java.util.List;
import java.util.Optional;
import java.util.Map;
import java.util.Objects;
import java.util.stream.Collectors;
import com.swingtrade.domain.SignalStrategyMetadata;

@Service
public class SignalStoreImpl implements SignalStore {

    private final SignalRepository repository;

    public SignalStoreImpl(SignalRepository repository) {
        this.repository = repository;
    }

    @Override
    public List<Signal> findAll() {
        return repository.findAll().stream()
            .map(SignalEntity::toDomain)
            .toList();
    }

    @Override
    public List<Signal> findBySymbol(String symbol) {
        return repository.findBySymbolOrderByDateDesc(symbol, org.springframework.data.domain.PageRequest.of(0, Integer.MAX_VALUE)).stream()
            .map(SignalEntity::toDomain)
            .toList();
    }

    @Override
    public List<Signal> findBySymbolOrderByDateDesc(String symbol) {
        return repository.findBySymbolOrderByDateDesc(symbol, org.springframework.data.domain.PageRequest.of(0, Integer.MAX_VALUE)).stream()
            .map(SignalEntity::toDomain)
            .toList();
    }

    @Override
    public List<Signal> findByDateAndStrategy(LocalDate date, String strategy) {
        return repository.findByDateAndStrategy(date, strategy).stream()
            .map(SignalEntity::toDomain)
            .toList();
    }

    @Override
    public List<Signal> findBySymbolAndDate(String symbol, LocalDate date) {
        return repository.findBySymbolAndDate(symbol, date).stream()
            .map(SignalEntity::toDomain)
            .toList();
    }

    @Override
    public List<String> findStrategiesBySymbolAndDate(String symbol, LocalDate date) {
        return repository.findStrategiesBySymbolAndDate(symbol, date);
    }

    @Override
    public Optional<String> findStrategyById(Long signalId) {
        return signalId == null ? Optional.empty() : repository.findStrategyById(signalId);
    }

    @Override
    public Map<Long, SignalStrategyMetadata> findStrategyMetadataByIds(List<Long> signalIds) {
        if (signalIds == null || signalIds.isEmpty()) return Map.of();
        return repository.findAllById(signalIds.stream().filter(Objects::nonNull).distinct().toList()).stream()
                .filter(entity -> entity.getId() != null)
                .collect(Collectors.toMap(SignalEntity::getId,
                        entity -> new SignalStrategyMetadata(entity.getStrategy(), entity.getStrategyVersion()),
                        (left, right) -> left));
    }

    @Override
    public List<Signal> findByType(Signal.SignalType type) {
        return repository.findBySignalType(
                type.name(), org.springframework.data.domain.PageRequest.of(0, Integer.MAX_VALUE)).stream()
            .map(SignalEntity::toDomain)
            .toList();
    }

    @Override
    public List<Signal> findUnprocessed() {
        return repository.findUnprocessedBuySignalsSince(LocalDate.now().minusDays(30)).stream()
            .map(SignalEntity::toDomain)
            .toList();
    }

    @Override
    public Signal save(Signal signal) {
        return repository.save(SignalEntity.fromDomain(signal)).toDomain();
    }

    @Override
    public Signal save(Signal signal, String warningFlag) {
        return repository.save(SignalEntity.fromDomain(signal, warningFlag)).toDomain();
    }

    @Override
    public Signal save(Signal signal, String warningFlag, String strategy) {
        return save(signal, warningFlag, strategy, 1);
    }

    @Override
    public Signal save(Signal signal, String warningFlag, String strategy, Integer strategyVersion) {
        return repository.save(SignalEntity.fromDomain(signal, warningFlag, strategy, strategyVersion)).toDomain();
    }

    @Override
    public Signal save(Signal signal, String warningFlag, String strategy, Integer strategyVersion,
                       String paramsHash) {
        SignalEntity entity = SignalEntity.fromDomain(signal, warningFlag, strategy, strategyVersion);
        entity.setParamsHash(paramsHash);
        return repository.save(entity).toDomain();
    }

    @Override
    public void markProcessed(Long signalId) {
        Optional<SignalEntity> entity = repository.findById(signalId);
        entity.ifPresent(e -> {
            e.setProcessed(true);
            repository.save(e);
        });
    }

    @Override
    public Optional<Signal> findLatestBySymbol(String symbol) {
        List<SignalEntity> results = repository.findLatestBySymbol(symbol,
            org.springframework.data.domain.PageRequest.of(0, 1));
        return results.stream().findFirst().map(SignalEntity::toDomain);
    }

    @Override
    public Optional<Signal> findLatestBySymbolAndStrategy(String symbol, String strategy) {
        return repository.findLatestBySymbolAndStrategy(symbol, strategy,
            org.springframework.data.domain.PageRequest.of(0, 1)).stream()
            .findFirst().map(SignalEntity::toDomain);
    }

    @Override
    public List<Signal> findLatestSignalPerSymbol() {
        return repository.findLatestSignalPerSymbol().stream()
            .map(SignalEntity::toDomain)
            .toList();
    }

    @Override
    public List<String> findAllDistinctSymbols() {
        return repository.findAllDistinctSymbols();
    }

    @Override
    public long countBySymbolAndDate(String symbol, LocalDate date) {
        return repository.countBySymbolAndDate(symbol, date);
    }

    @Override
    public List<Signal> findBuySignalsSince(LocalDate sinceDate) {
        return repository.findBuySignalsSince(sinceDate,
                org.springframework.data.domain.PageRequest.of(0, Integer.MAX_VALUE)).stream()
            .map(SignalEntity::toDomain)
            .toList();
    }

    @Override
    public List<Signal> findByDateRange(LocalDate startDate, LocalDate endDate) {
        return repository.findByDateRange(
                startDate, endDate,
                org.springframework.data.domain.PageRequest.of(0, Integer.MAX_VALUE)).stream()
            .map(SignalEntity::toDomain)
            .toList();
    }

    @Override
    public List<Signal> findByDateRangeAndType(LocalDate startDate, LocalDate endDate, Signal.SignalType type) {
        return repository.findByDateRangeAndSignalType(
                startDate, endDate, type.name(),
                org.springframework.data.domain.PageRequest.of(0, Integer.MAX_VALUE)).stream()
            .map(SignalEntity::toDomain)
            .toList();
    }

    @Override
    public List<Signal> findByMinConfidence(double minConfidence) {
        return repository.findByMinConfidence(
                BigDecimal.valueOf(minConfidence),
                org.springframework.data.domain.PageRequest.of(0, Integer.MAX_VALUE)).stream()
            .map(SignalEntity::toDomain)
            .toList();
    }

    @Override
    @Transactional
    public int deleteBySymbolAndDate(String symbol, LocalDate date) {
        return repository.deleteBySymbolAndDate(symbol, date);
    }

    @Override
    @Transactional
    public int deleteBySymbolAndDateAndStrategy(String symbol, LocalDate date, String strategy) {
        return repository.deleteBySymbolAndDateAndStrategy(symbol, date, strategy);
    }

    @Override
    @Transactional
    public int deleteByDate(LocalDate date) {
        return repository.deleteByDate(date);
    }

    @Override
    @Transactional
    public int deleteAllSignals() {
        return repository.deleteAllSignals();
    }
}
