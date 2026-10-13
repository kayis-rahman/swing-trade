package com.swingtrade.data.repository;

import com.swingtrade.data.entity.IntradayPaperSessionEntity;
import org.springframework.data.jpa.repository.JpaRepository;

import java.util.List;
import java.util.Optional;

public interface IntradayPaperSessionRepository extends JpaRepository<IntradayPaperSessionEntity, String> {
    Optional<IntradayPaperSessionEntity> findByRunId(String runId);
    List<IntradayPaperSessionEntity> findTop50ByOrderByCreatedAtDesc();
}
