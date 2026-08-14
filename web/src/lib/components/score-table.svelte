<script lang="ts">
  import { scoreTier } from "$lib/admission-blocks";
  import { IDENTITY_COLUMNS, SUBJECTS, hasAnyValue } from "$lib/subjects";
  import type { Student } from "$lib/types";

  let { results }: { results: Student[] | null } = $props();

  function formatScore(val: number | null): string {
    return val === null || val === undefined ? "—" : Number(val).toFixed(2);
  }

  // Drop columns that are NULL for every row in the result set. That is what
  // lets one table serve both exam years with no dataset conditional here: each
  // year's unused columns simply drop out.
  const visibleIdentity = $derived(
    results ? IDENTITY_COLUMNS.filter((col) => hasAnyValue(results, col.key)) : [],
  );
  const visibleSubjects = $derived(
    results ? SUBJECTS.filter((col) => hasAnyValue(results, col.key)) : [],
  );
</script>

{#if results && results.length === 0}
  <p class="notice bg-surface-alt text-ink-muted">Không tìm thấy kết quả.</p>
{:else if results}
  <div class="mt-4 overflow-x-auto">
    <p class="mb-2 text-sm text-ink-muted">Tìm thấy {results.length} kết quả</p>
    <table class="w-full border-collapse bg-surface text-sm max-sm:text-xs">
      <thead>
        <tr
          class="[&>th]:sticky [&>th]:top-0 [&>th]:border-b-2 [&>th]:border-line-strong
                 [&>th]:bg-surface-alt [&>th]:px-2 [&>th]:py-2.5 [&>th]:text-left
                 [&>th]:whitespace-nowrap"
        >
          <th>SBD</th>
          <th>Họ tên</th>
          <th>Ngày sinh</th>
          {#each visibleIdentity as col (col.key)}
            <th>{col.label}</th>
          {/each}
          {#each visibleSubjects as col (col.key)}
            <th>{col.label}</th>
          {/each}
        </tr>
      </thead>
      <tbody>
        {#each results as row (row.so_bao_danh)}
          <tr class="hover:bg-surface-alt [&>td]:border-b [&>td]:border-line [&>td]:px-2 [&>td]:py-2">
            <td class="font-mono text-xs whitespace-nowrap text-ink-muted">{row.so_bao_danh}</td>
            <td class="whitespace-nowrap">{row.ho_ten}</td>
            <td>{row.ngay_sinh || "—"}</td>
            {#each visibleIdentity as col (col.key)}
              <td
                class={col.key === "ten_cum_thi"
                  ? "max-w-[220px] truncate text-xs text-ink-muted"
                  : undefined}
                title={col.key === "ten_cum_thi" ? (row[col.key] ?? "") : undefined}
              >
                {row[col.key] || "—"}
              </td>
            {/each}
            {#each visibleSubjects as col (col.key)}
              {@const tier = scoreTier(row[col.key])}
              <td
                class="text-center font-medium tabular-nums {tier
                  ? `tier-surface tier-${tier.key}`
                  : ''}"
                title={tier?.label}
              >
                {formatScore(row[col.key])}
              </td>
            {/each}
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
{/if}
