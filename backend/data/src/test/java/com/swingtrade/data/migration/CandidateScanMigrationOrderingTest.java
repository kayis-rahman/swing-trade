package com.swingtrade.data.migration;

import org.flywaydb.core.api.MigrationVersion;
import org.junit.jupiter.api.Test;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.List;
import java.util.Map;
import java.util.stream.Collectors;

import static org.assertj.core.api.Assertions.assertThat;

class CandidateScanMigrationOrderingTest {
    private static final Path MIGRATION_DIRECTORY = Path.of("src/main/resources/db/migration");
    private static final String CANDIDATE_SCAN_MIGRATION = "candidate_scan_trigger";

    @Test
    void candidateScanTriggerMigrationKeepsItsAssignedVersion() throws IOException {
        Map<String, MigrationVersion> versions;
        try (var migrationFiles = Files.list(MIGRATION_DIRECTORY)) {
            versions = migrationFiles
                .map(path -> path.getFileName().toString())
                .filter(name -> name.startsWith("V") && name.contains("__"))
                .collect(Collectors.toMap(
                    name -> name,
                    CandidateScanMigrationOrderingTest::versionFromFilename));
        }

        List<MigrationVersion> candidateScanVersions = versions.entrySet().stream()
            .filter(entry -> entry.getKey().contains("__" + CANDIDATE_SCAN_MIGRATION + ".sql"))
            .map(Map.Entry::getValue)
            .toList();
        assertThat(candidateScanVersions).containsExactly(MigrationVersion.fromVersion("76"));
    }

    private static MigrationVersion versionFromFilename(String filename) {
        int separator = filename.indexOf("__");
        return MigrationVersion.fromVersion(filename.substring(1, separator));
    }
}
