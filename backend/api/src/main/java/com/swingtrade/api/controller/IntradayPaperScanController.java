package com.swingtrade.api.controller;

import com.swingtrade.api.dto.IntradayPaperScanRequest;
import com.swingtrade.broker.service.intraday.IntradayPaperScanResult;
import com.swingtrade.broker.service.intraday.IntradayPaperScanService;
import jakarta.validation.Valid;
import io.github.resilience4j.ratelimiter.annotation.RateLimiter;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

/** Read-only retrospective scanner over stored watchlist bars; never contacts a broker or provider. */
@RestController
@RequestMapping("/api/intraday/paper/scans")
public class IntradayPaperScanController {
    private final IntradayPaperScanService scans;

    public IntradayPaperScanController(IntradayPaperScanService scans) {
        this.scans = scans;
    }

    @PostMapping
    @RateLimiter(name = "candidateScan")
    public ResponseEntity<IntradayPaperScanResult> scan(@Valid @RequestBody IntradayPaperScanRequest request) {
        try {
            return ResponseEntity.ok(scans.scan(request.sessionDate(), request.asOfBarTime(), request.resolvedRiskConfig()));
        } catch (IllegalArgumentException e) {
            return ResponseEntity.badRequest().build();
        } catch (IllegalStateException e) {
            return ResponseEntity.status(org.springframework.http.HttpStatus.UNPROCESSABLE_ENTITY).build();
        }
    }
}
