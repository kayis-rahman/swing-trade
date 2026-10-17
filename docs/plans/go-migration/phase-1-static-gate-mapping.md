# Phase 1 static-gate mapping

This maps the configured Java Checkstyle/PMD policies to Go checks for the platform skeleton. Java rules are sourced from `backend/config/checkstyle/checkstyle.xml` and `backend/config/pmd/pmd-ruleset.xml`. This is a Go implementation mapping, not a claim that G0 passed; G0 was waived by captain decision. No Java gates or Java code are changed.

## Checkstyle

| Java Checkstyle module | Go equivalent or reason |
|---|---|
| `ClassTypeParameterName`, `MethodTypeParameterName`, `InterfaceTypeParameterName` | `revive` naming rules plus Go compiler type-parameter validation; Go has no class hierarchy. |
| `PackageName` | `revive` `var-naming` package checks and valid Go import paths. |
| `ParameterName`, `LocalFinalVariableName`, `LocalVariableName`, `MemberName`, `MethodName`, `TypeName` | Go naming convention enforced by `revive` `var-naming`/`exported`; compiler enforces identifier syntax. Go has no final locals or instance fields. |
| `AvoidStarImport` | No equivalent needed: Go imports are package-qualified and have no wildcard-import syntax. |
| `IllegalImport` | `depguard` import allow/deny lists plus Go module boundaries. |
| `RedundantImport`, `UnusedImports` | `gofmt`/`goimports` and compiler unused-import errors. |
| `IllegalTokenText` | Go parser/compiler rejects invalid tokens; `gofmt` canonicalizes source. |
| `ModifierOrder` | No equivalent: Go has no Java-style modifier list; declaration grammar fixes order. |
| `OneStatementPerLine` | `gofmt` and idiomatic Go style; Go's parser permits semicolons, but no line-count linter is enabled. |
| `UnnecessaryParentheses` | `gocritic` checks where available; parser and compiler preserve equivalent expression semantics. |
| `HideUtilityClassConstructor` | No equivalent: Go has package-level functions and no static utility classes or constructors. |
| `InnerTypeLast` | No equivalent: Go types are package-level; declaration order does not affect visibility or resolution. |
| `InterfaceIsType` | No equivalent: Go interfaces are types by language definition. |
| `VisibilityModifier` | Go capitalization controls export; `revive` `exported` checks exported identifiers. |
| `MethodLength` (`max=150`) | `funlen` with a 150-line ceiling. |
| `FileLength` (`max=1400`) | No direct Go standard-linter equivalent; kept as a documented review constraint rather than a brittle source-text gate. |

## PMD

The ruleset enables PMD 7 `category/java/errorprone.xml`, `category/java/security.xml`, and `category/java/multithreading.xml`. It explicitly excludes these Java-specific/noisy checks: `AvoidCatchingGenericException`, `AvoidDuplicateLiterals`, `AvoidLiteralsInIfCondition`, `NullAssignment`, `TestClassWithoutTestCases`, `MissingSerialVersionUID`, `AvoidFieldNameMatchingMethodName`, `UseLocaleWithCaseConversions`, `CloseResource`, `AssignmentInOperand`, `IdenticalConditionalBranches`, `EmptyCatchBlock`, `UseProperClassLoader`, `AvoidSynchronizedStatement`, `AvoidSynchronizedAtMethodLevel`, `UseConcurrentHashMap`, `DoNotUseThreads`, and `AvoidUsingVolatile`.

| Effective PMD category | Go equivalent or reason |
|---|---|
| `errorprone` (after the 14 listed exclusions) | `staticcheck`, `go vet`, `gocritic`, `ineffassign`, `unused`, `errcheck`, `nilerr`; language/compiler checks cover Java-specific constructs that do not exist in Go. |
| `security` | `gosec`; `govulncheck` covers known vulnerable dependencies. |
| `multithreading` (after the five listed exclusions) | Go race detector (`go test -race`), `go vet`, `staticcheck`, and targeted concurrency tests; Java monitor/thread APIs excluded by PMD have no Go equivalent. |

## Enforced Go toolchain

`gofmt` and `goimports`; `golangci-lint` with `revive`, `staticcheck`, `govet`, `errcheck`, `gocritic`, `gosec`, `ineffassign`, `unused`, `bodyclose`, `sqlclosecheck`, `nilerr`, `depguard`, `forbidigo` and `funlen`; `go test -race`; `govulncheck`; and workspace architecture checks. Money packages forbid `float64`; `time.Now` is allowed only behind the clock port. Tools that are not installed are reported as unavailable, never silently counted green.
