<template>
  <section class="mt-6 card-panel p-5" data-testid="strategy-matrix">
    <h3 class="mb-1 text-sm font-semibold text-text-primary">Strategy matrix</h3>
    <p class="mb-3 text-xs text-text-muted">
      Per-symbol outcome of every strategy in the SIGNAL stage
    </p>
    <div class="overflow-x-auto">
      <table class="w-full min-w-[480px] border-collapse text-sm">
        <thead>
          <tr class="border-b border-border-subtle text-left text-xs text-text-muted">
            <th class="pb-2 pr-4 font-medium">Symbol</th>
            <th v-for="v in matrix.variants" :key="v.variantId" class="pb-2 pr-4 font-medium">
              {{ v.variantId }} <span class="text-[10px]">v{{ v.version }}</span>
            </th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="symbol in matrix.symbols"
            :key="symbol"
            class="border-b border-border-subtle/50"
          >
            <td class="py-2 pr-4 font-medium text-text-primary">{{ symbol }}</td>
            <td
              v-for="v in matrix.variants"
              :key="v.variantId"
              class="py-2 pr-4 align-top"
              :data-testid="`cell-${symbol}-${v.variantId}`"
              :data-kind="kind(symbol, v.variantId)"
            >
              <span :class="kindClass(kind(symbol, v.variantId))">{{
                label(symbol, v.variantId)
              }}</span>
              <p v-if="reason(symbol, v.variantId)" class="mt-0.5 text-[11px] text-text-muted">
                {{ reason(symbol, v.variantId) }}
              </p>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { JobRunStageResponse } from '../api/types'
import { buildStrategyMatrix, matrixCellKind, type MatrixCellKind } from '../utils/runMatrix'

const props = defineProps<{ rows: JobRunStageResponse[] }>()
const matrix = computed(() => buildStrategyMatrix(props.rows))

const kind = (symbol: string, variantId: string): MatrixCellKind =>
  matrixCellKind(matrix.value.cell(symbol, variantId))

function label(symbol: string, variantId: string): string {
  const cell = matrix.value.cell(symbol, variantId)
  switch (kind(symbol, variantId)) {
    case 'signal':
      return `${cell?.signal === true ? 'Signal' : (cell?.signal ?? '')}${cell?.score != null ? ` (${cell.score})` : ''}`
    case 'no-signal':
      return 'Evaluated · no signal'
    case 'skipped':
      return 'Skipped'
    case 'error':
      return 'Error'
    default:
      return '—'
  }
}

const reason = (symbol: string, variantId: string): string | null =>
  matrix.value.cell(symbol, variantId)?.reason ?? null

function kindClass(k: MatrixCellKind): string {
  return (
    {
      signal: 'font-medium text-success',
      'no-signal': 'text-text-muted',
      skipped: 'font-medium text-warning',
      error: 'font-medium text-danger',
      none: 'text-text-muted',
    } as Record<MatrixCellKind, string>
  )[k]
}
</script>
