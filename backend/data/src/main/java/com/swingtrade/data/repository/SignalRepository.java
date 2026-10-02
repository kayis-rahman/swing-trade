package com.swingtrade.data.repository;

import com.swingtrade.data.entity.SignalEntity;
import org.springframework.data.domain.Pageable;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Modifying;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;
import org.springframework.transaction.annotation.Transactional;

import java.math.BigDecimal;
import java.time.LocalDate;
import java.util.List;

/**
 * Repository interface for SignalEntity operations.
 */
@Repository
public interface SignalRepository extends JpaRepository<SignalEntity, Long> {

    /**
     * Finds the latest signals for a stock.
     *
     * @param symbol the stock symbol
     * @param pageable pagination
     * @return list of signals
     */
    @Query("SELECT s FROM SignalEntity s WHERE s.symbol = :symbol ORDER BY s.date DESC, s.createdAt DESC, s.id DESC")
    List<SignalEntity> findBySymbolOrderByDateDesc(
        @Param("symbol") String symbol,
        Pageable pageable
    );

    /**
     * Finds all signals for a stock on a specific date.
     *
     * @param symbol the stock symbol
     * @param date the date
     * @return list of signals
     */
    @Query("SELECT s FROM SignalEntity s WHERE s.symbol = :symbol AND s.date = :date")
    List<SignalEntity> findBySymbolAndDate(
        @Param("symbol") String symbol,
        @Param("date") LocalDate date
    );

    @Query("SELECT DISTINCT s.strategy FROM SignalEntity s WHERE s.symbol = :symbol AND s.date = :date")
    List<String> findStrategiesBySymbolAndDate(
        @Param("symbol") String symbol,
        @Param("date") LocalDate date
    );

    /**
     * Finds all BUY signals generated on or after a date.
     *
     * @param date the date
     * @return list of BUY signals
     */
    @Query("SELECT s FROM SignalEntity s WHERE s.signalType = 'BUY' AND s.date >= :date ORDER BY s.date DESC, s.createdAt DESC")
    List<SignalEntity> findBuySignalsSince(
        @Param("date") LocalDate date,
        Pageable pageable
    );

    /**
     * Finds signals by date range and signal type.
     *
     * @param startDate the start date (inclusive)
     * @param endDate the end date (inclusive)
     * @param signalType the signal type (BUY, SELL, HOLD)
     * @param pageable pagination
     * @return list of signals
     */
    @Query("SELECT s FROM SignalEntity s WHERE s.date BETWEEN :startDate AND :endDate AND s.signalType = :signalType ORDER BY s.date DESC")
    List<SignalEntity> findByDateRangeAndSignalType(
        @Param("startDate") LocalDate startDate,
        @Param("endDate") LocalDate endDate,
        @Param("signalType") String signalType,
        Pageable pageable
    );

    @Query("SELECT s FROM SignalEntity s WHERE s.signalType = :signalType ORDER BY s.date DESC")
    List<SignalEntity> findBySignalType(
        @Param("signalType") String signalType,
        Pageable pageable
    );

    /**
     * Finds all signals within a date range regardless of signal type.
     *
     * @param startDate start date (inclusive)
     * @param endDate end date (inclusive)
     * @param pageable pagination
     * @return list of all signals in the date range
     */
    @Query("SELECT s FROM SignalEntity s WHERE s.date BETWEEN :startDate AND :endDate ORDER BY s.date DESC")
    List<SignalEntity> findByDateRange(
        @Param("startDate") LocalDate startDate,
        @Param("endDate") LocalDate endDate,
        Pageable pageable
    );

    /**
     * Finds signals with confidence score at or above the threshold.
     *
     * @param minConfidence minimum confidence score
     * @param pageable pagination
     * @return list of signals meeting the confidence threshold
     */
    @Query("SELECT s FROM SignalEntity s WHERE s.confidenceScore >= :minConfidence ORDER BY s.confidenceScore DESC, s.date DESC")
    List<SignalEntity> findByMinConfidence(
        @Param("minConfidence") BigDecimal minConfidence,
        Pageable pageable
    );

    /**
     * Finds the latest signal for a stock.
     *
     * @param symbol the stock symbol
     * @return optional containing the latest signal
     */
    @Query("SELECT s FROM SignalEntity s WHERE s.symbol = :symbol ORDER BY s.date DESC, s.createdAt DESC")
    List<SignalEntity> findLatestBySymbol(@Param("symbol") String symbol, Pageable pageable);

    /**
     * Counts signals by date and symbol.
     *
     * @param symbol the stock symbol
     * @param date the date
     * @return the count
     */
    @Query("SELECT COUNT(s) FROM SignalEntity s WHERE s.symbol = :symbol AND s.date = :date")
    long countBySymbolAndDate(
        @Param("symbol") String symbol,
        @Param("date") LocalDate date
    );

    /**
     * Deletes signals older than a specific date.
     *
     * @param cutoffDate the cutoff date
     * @return number of deleted signals
     */
    @Query("DELETE FROM SignalEntity s WHERE s.date < :cutoffDate")
    int deleteSignalsBefore(@Param("cutoffDate") LocalDate cutoffDate);

    /**
     * Finds all distinct symbols with signals.
     *
     * @return list of distinct symbols
     */
    @Query("SELECT DISTINCT s.symbol FROM SignalEntity s ORDER BY s.symbol")
    List<String> findAllDistinctSymbols();

    /**
     * Finds all signals for a stock on a specific date generated by a specific strategy.
     * Used to dedupe signal generation per strategy per day.
     *
     * @param symbol   the stock symbol
     * @param date     the date
     * @param strategy the strategy identifier (e.g. "DEFAULT", "PRICE_ACTION")
     * @return list of signals
     */
    @Query("SELECT s FROM SignalEntity s WHERE s.symbol = :symbol AND s.date = :date AND s.strategy = :strategy")
    List<SignalEntity> findBySymbolAndDateAndStrategy(
        @Param("symbol") String symbol,
        @Param("date") LocalDate date,
        @Param("strategy") String strategy
    );

    @Query("SELECT s.id FROM SignalEntity s WHERE s.symbol = :symbol AND s.date = :date AND s.strategy = :strategy")
    List<Long> findIdsBySymbolAndDateAndStrategy(
        @Param("symbol") String symbol, @Param("date") LocalDate date, @Param("strategy") String strategy);

    @Query("SELECT s.strategy FROM SignalEntity s WHERE s.id = :signalId")
    java.util.Optional<String> findStrategyById(@Param("signalId") Long signalId);

    /**
     * Finds every signal recorded for a specific date and variant, regardless of symbol - used by
     * strategy attribution to read what the SIGNAL stage persisted for a variant on a given day.
     */
    @Query("SELECT s FROM SignalEntity s WHERE s.date = :date AND s.strategy = :strategy")
    List<SignalEntity> findByDateAndStrategy(
        @Param("date") LocalDate date,
        @Param("strategy") String strategy
    );

    /**
     * Finds the latest signal for a stock generated by a specific strategy.
     *
     * @param symbol   the stock symbol
     * @param strategy the strategy identifier
     * @return optional containing the latest signal
     */
    @Query("SELECT s FROM SignalEntity s WHERE s.symbol = :symbol AND s.strategy = :strategy ORDER BY s.date DESC, s.createdAt DESC")
    List<SignalEntity> findLatestBySymbolAndStrategy(
        @Param("symbol") String symbol,
        @Param("strategy") String strategy,
        Pageable pageable
    );

    /**
     * Finds unprocessed BUY signals from the last N minutes.
     *
     * @param since the cutoff time
     * @return list of unprocessed BUY signals
     */
    @Query("SELECT s FROM SignalEntity s WHERE s.signalType = 'BUY' AND s.date >= :since AND s.processed = false ORDER BY s.date ASC")
    List<SignalEntity> findUnprocessedBuySignalsSince(
        @Param("since") LocalDate since
    );

    /**
     * Deletes all signals for a specific symbol and date.
     */
    @Modifying
    @Transactional
    @Query("DELETE FROM SignalEntity s WHERE s.symbol = :symbol AND s.date = :date")
    int deleteBySymbolAndDate(@Param("symbol") String symbol, @Param("date") LocalDate date);

    @Modifying
    @Transactional
    @Query("DELETE FROM SignalEntity s WHERE s.symbol = :symbol AND s.date = :date AND s.strategy = :strategy")
    int deleteBySymbolAndDateAndStrategy(@Param("symbol") String symbol, @Param("date") LocalDate date,
                                         @Param("strategy") String strategy);

    /**
     * Deletes all signals for a specific date.
     */
    @Modifying
    @Transactional
    @Query("DELETE FROM SignalEntity s WHERE s.date = :date")
    int deleteByDate(@Param("date") LocalDate date);

    /**
     * Deletes all signals.
     */
    @Modifying
    @Transactional
    @Query("DELETE FROM SignalEntity")
    int deleteAllSignals();

    /**
     * Finds the single latest signal per symbol across the whole table, using a DB-level
     * window function so no more than one row per symbol is ever loaded into memory.
     * Ties (same date) are broken by created_at then id, matching the ordering used by
     * {@link #findLatestBySymbol(String, Pageable)}.
     *
     * @return one signal per distinct symbol, the most recent for that symbol
     */
    @Query(value = "SELECT ranked.* FROM (" +
        "  SELECT s.*, ROW_NUMBER() OVER (" +
        "    PARTITION BY s.symbol ORDER BY s.date DESC, s.created_at DESC, s.id DESC" +
        "  ) AS rn " +
        "  FROM signals s" +
        ") ranked WHERE ranked.rn = 1",
        nativeQuery = true)
    List<SignalEntity> findLatestSignalPerSymbol();
}
