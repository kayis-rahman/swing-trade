package com.swingtrade.data.repository;

import com.swingtrade.data.entity.SentimentClassificationLogEntity;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.stereotype.Repository;

import java.time.LocalDate;
import java.util.List;

@Repository
public interface SentimentClassificationLogRepository
        extends JpaRepository<SentimentClassificationLogEntity, Long> {

    List<SentimentClassificationLogEntity> findByAnalysisDateBetweenAndShadowModeTrue(
            LocalDate startDate, LocalDate endDate);
}
