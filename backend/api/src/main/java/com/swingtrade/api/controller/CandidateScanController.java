package com.swingtrade.api.controller;

import com.swingtrade.api.service.CandidateScanService;
import com.swingtrade.api.scheduler.CandidateScanHandoffDispatcher;
import com.swingtrade.data.entity.CandidateScanRunEntity;
import io.github.resilience4j.ratelimiter.annotation.RateLimiter;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.PutMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.servlet.mvc.method.annotation.SseEmitter;

import java.util.List;
import java.util.Map;
import java.util.UUID;

@RestController
@RequestMapping("/api/candidate-scans")
public class CandidateScanController {
    private final CandidateScanService service;
    private final CandidateScanHandoffDispatcher handoffs;

    public CandidateScanController(CandidateScanService service, CandidateScanHandoffDispatcher handoffs) {
        this.service = service;
        this.handoffs = handoffs;
    }

    @GetMapping("/settings")
    public ResponseEntity<?> getSettings() {
        return ResponseEntity.ok(service.getScanSettings());
    }

    @PutMapping("/settings")
    public ResponseEntity<?> updateSettings(@RequestBody Map<String, String> body) {
        try {
            return ResponseEntity.ok(service.updateScanSettings(body));
        } catch (IllegalArgumentException e) {
            return ResponseEntity.badRequest().body(Map.of("error", e.getMessage()));
        }
    }

    @PostMapping
    @RateLimiter(name = "candidateScan")
    public ResponseEntity<?> start() {
        try { return ResponseEntity.ok(service.start()); }
        catch (IllegalStateException e) {
            return ResponseEntity.status(HttpStatus.CONFLICT).body(Map.of("error", e.getMessage()));
        }
    }

    @GetMapping
    public List<CandidateScanRunEntity> history() { return service.getHistory(); }

    @GetMapping("/{runId}")
    public ResponseEntity<?> run(@PathVariable UUID runId) {
        CandidateScanRunEntity run = service.getRun(runId);
        return run == null ? ResponseEntity.notFound().build() : ResponseEntity.ok(run);
    }

    @GetMapping("/{runId}/results")
    public ResponseEntity<?> results(
            @PathVariable UUID runId,
            @RequestParam(defaultValue = "0") int offset,
            @RequestParam(defaultValue = "100") int limit,
            @RequestParam(defaultValue = "") String symbol,
            @RequestParam(defaultValue = "") String signalType) {
        if (service.getRun(runId) == null) return ResponseEntity.notFound().build();
        return ResponseEntity.ok(service.getResultsPage(runId, offset, limit, symbol, signalType));
    }

    @GetMapping("/{runId}/stream")
    public ResponseEntity<SseEmitter> stream(@PathVariable UUID runId) {
        SseEmitter emitter = service.stream(runId);
        return emitter == null ? ResponseEntity.notFound().build() : ResponseEntity.ok(emitter);
    }

    @PostMapping("/{runId}/analysis")
    public ResponseEntity<?> startAnalysis(@PathVariable UUID runId, @RequestBody AnalysisScopeRequest request) {
        try {
            return ResponseEntity.ok(handoffs.startManualAnalysis(runId, request.scope(), request.symbols()));
        } catch (IllegalArgumentException e) {
            return ResponseEntity.badRequest().body(Map.of("error", e.getMessage()));
        } catch (IllegalStateException e) {
            return ResponseEntity.status(HttpStatus.CONFLICT).body(Map.of("error", e.getMessage()));
        }
    }

    public record AnalysisScopeRequest(String scope, List<String> symbols) { }

    @PostMapping("/{runId}/cancel")
    public ResponseEntity<?> cancel(@PathVariable UUID runId) {
        if (service.getRun(runId) == null) return ResponseEntity.notFound().build();
        service.cancel(runId);
        return ResponseEntity.ok(service.getRun(runId));
    }

    @PostMapping("/{runId}/pause")
    public ResponseEntity<?> pause(@PathVariable UUID runId) {
        if (service.getRun(runId) == null) return ResponseEntity.notFound().build();
        if (!service.pause(runId)) return ResponseEntity.badRequest().body(Map.of("error", "Scan is not running"));
        return ResponseEntity.ok(service.getRun(runId));
    }

    @PostMapping("/{runId}/resume")
    public ResponseEntity<?> resume(@PathVariable UUID runId) {
        if (service.getRun(runId) == null) return ResponseEntity.notFound().build();
        if (!service.resume(runId)) return ResponseEntity.badRequest().body(Map.of("error", "Scan is not paused"));
        return ResponseEntity.ok(service.getRun(runId));
    }
}
