# ==============================================================================
# SwingTrade API - Stage Runtime Image
# ==============================================================================
# The JAR + runtime deps are built OUTSIDE Docker, on the Mac, from a git
# worktree checked out to the `stage` branch (see dev-stack.sh `stage`).
# Building Gradle on pi-node's constrained hardware was too slow; this image
# just packages an already-built lib/ directory. Multi-stage keeps the build
# context (COPY source) out of the final image and lets Docker cache the
# `artifacts` layer independently of the runtime base image layer.
# ==============================================================================

# ------------------------------------------------------------------------------
# Stage 1: artifacts - stage the prebuilt jar + runtime deps
# ------------------------------------------------------------------------------
FROM scratch AS artifacts
COPY lib/ /lib/

# ------------------------------------------------------------------------------
# Stage 2: runtime-stage - stage deployment image (clean, no debug)
# ------------------------------------------------------------------------------
FROM eclipse-temurin:21-jre-alpine AS runtime-stage

ARG BUILD_COMMIT=unknown
LABEL org.opencontainers.image.revision="${BUILD_COMMIT}" \
      org.opencontainers.image.source="swing-trade"

WORKDIR /app

COPY --from=artifacts /lib/ ./lib/

# Set timezone to Asia/Kolkata for correct scheduling
RUN apk add --no-cache openssh-client && \
    ln -sf /usr/share/zoneinfo/Asia/Kolkata /etc/localtime && \
    echo "Asia/Kolkata" > /etc/timezone

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=10s --start-period=120s --retries=3 \
    CMD wget -qO- http://localhost:8080/actuator/health || exit 1

# Startup script builds classpath from all JARs in lib/
RUN printf '#!/bin/sh\n' > /app/start.sh && \
    printf 'CP=""\n' >> /app/start.sh && \
    printf 'for f in /app/lib/*.jar; do\n' >> /app/start.sh && \
    printf '  CP="${CP:+$CP:}$f"\n' >> /app/start.sh && \
    printf 'done\n' >> /app/start.sh && \
    printf 'exec java -cp "$CP" -Duser.timezone=Asia/Kolkata -Dspring.profiles.active=${SPRING_PROFILES_ACTIVE:-stage} -Dserver.port=${SERVER_PORT:-8080} com.swingtrade.api.app.SwingTradeApiApplication\n' >> /app/start.sh && \
    chmod +x /app/start.sh

CMD ["/app/start.sh"]
