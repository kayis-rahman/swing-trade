package com.swingtrade.data.repository;

import com.swingtrade.data.entity.StockEntity;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;

import java.util.List;
import java.util.Optional;

/**
 * Repository interface for StockEntity operations.
 */
@Repository
public interface StockRepository extends JpaRepository<StockEntity, Long> {

    /**
     * Finds a stock by its symbol.
     *
     * @param symbol the stock symbol
     * @return optional containing the stock if found
     */
    Optional<StockEntity> findBySymbol(String symbol);

    /**
     * Checks if a stock with the given symbol exists.
     *
     * @param symbol the stock symbol
     * @return true if the stock exists
     */
    boolean existsBySymbol(String symbol);

    /**
     * Finds all stocks in a specific exchange.
     *
     * @param exchange the exchange name (NSE or BSE)
     * @return list of stocks in the exchange
     */
    List<StockEntity> findByExchange(String exchange);

    /**
     * Finds all stocks in a specific sector.
     *
     * @param sector the sector name
     * @return list of stocks in the sector
     */
    List<StockEntity> findBySector(String sector);

    /**
     * Finds all stocks that were added on or before a specific date.
     *
     * @param addedOn the added on date
     * @return list of stocks added on or before the date
     */
    List<StockEntity> findByAddedOnBefore(java.time.LocalDate addedOn);

    /**
     * Counts the total number of stocks.
     *
     * @return the total count
     */
    long countBy();

    /**
     * Finds all stocks ordered by symbol.
     *
     * @return list of stocks ordered by symbol
     */
    List<StockEntity> findAllByOrderBySymbol();

    /**
     * Checks if a symbol exists in NSE or BSE.
     *
     * @param symbol the stock symbol
     * @return true if the symbol exists in any exchange
     */
    @Query("SELECT COUNT(s) > 0 FROM StockEntity s WHERE s.symbol = :symbol")
    boolean existsSymbol(@Param("symbol") String symbol);

    /**
     * Finds all distinct stock symbols.
     *
     * @return list of distinct symbols
     */
    @Query("SELECT DISTINCT s.symbol FROM StockEntity s ORDER BY s.symbol")
    List<String> findAllDistinctSymbols();
}
