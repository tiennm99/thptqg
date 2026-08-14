<script>
  import { formatBytes, isBudgetError } from "$lib/sqlite.svelte";

  const MAX_ROWS = 1000;
  const PLACEHOLDER = "Nhập truy vấn SQL...\nVí dụ: SELECT * FROM student WHERE toan >= 9 LIMIT 10";

  let { db, disabled = false, presets = [] } = $props();

  // By convention every preset list ends with a "Hệ thống" group whose first
  // query dumps the table schema. See lib/sql-presets.ts.
  const schemaPreset = $derived(presets[presets.length - 1]?.queries[0]);

  let sql = $state("");
  let columns = $state([]);
  let rows = $state([]);
  let queryError = $state(null);
  let execTime = $state(null);
  let running = $state(false);
  let autoRan = false;

  async function execute(queryStr) {
    const source = db;
    if (!source?.ready) return;
    queryError = null;
    columns = [];
    rows = [];
    execTime = null;

    const trimmed = queryStr.trim();
    if (!trimmed) return;

    // Read-only statements only. The database is remote and read-only anyway,
    // so this guards the user's own session against a typo.
    const upper = trimmed.toUpperCase();
    if (!["SELECT", "PRAGMA", "EXPLAIN", "WITH"].some((kw) => upper.startsWith(kw))) {
      queryError = "Chỉ hỗ trợ truy vấn đọc (SELECT, PRAGMA, EXPLAIN, WITH).";
      return;
    }

    // Cap an unbounded SELECT: the tables run to hundreds of thousands of rows
    // and rendering them all would lock the tab.
    let finalSql = trimmed;
    if (upper.startsWith("SELECT") && !upper.includes("LIMIT") && !upper.includes("PRAGMA")) {
      finalSql = `${trimmed.replace(/;$/, "")} LIMIT ${MAX_ROWS}`;
    }

    running = true;
    const start = performance.now();
    try {
      const result = await source.query(finalSql, [], "SQL tab");
      execTime = (performance.now() - start).toFixed(1);
      rows = result.slice(0, MAX_ROWS);
      columns = rows.length > 0 ? Object.keys(rows[0]) : [];
    } catch (err) {
      queryError = isBudgetError(err)
        ? "Truy vấn này phải đọc quá nhiều dữ liệu và đã bị dừng. Hãy thêm điều kiện lọc, " +
          "hoặc dùng cột đã có chỉ mục (so_bao_danh, toan, khtn, khxh, ten_cum_thi)."
        : err instanceof Error
          ? err.message
          : String(err);
    } finally {
      running = false;
    }
  }

  function runPreset(presetSql) {
    sql = presetSql;
    void execute(presetSql);
  }

  // Show the student columns the first time the tab opens, rather than a blank
  // textarea. Once only, however the database changes underneath.
  $effect(() => {
    if (!db?.ready || autoRan || !schemaPreset) return;
    autoRan = true;
    runPreset(schemaPreset.sql);
  });

</script>

<div class="mx-auto max-w-[900px]">
  <div class="mb-4 flex flex-col gap-2.5">
    {#each presets as group (group.category)}
      <div class="grid grid-cols-[140px_1fr] items-start gap-3 max-sm:grid-cols-1 max-sm:gap-1.5">
        <span
          class="pt-1 text-xs font-semibold tracking-wider whitespace-nowrap text-ink-muted uppercase"
        >
          {group.category}
        </span>
        <div class="flex flex-wrap gap-1.5">
          {#each group.queries as preset (preset.label)}
            <button
              type="button"
              class="btn-chip rounded-md"
              onclick={() => runPreset(preset.sql)}
              disabled={disabled || running}
            >
              {preset.label}
            </button>
          {/each}
        </div>
      </div>
    {/each}
  </div>

  <form
    class="query-form mb-4"
    onsubmit={(e) => {
      e.preventDefault();
      void execute(sql);
    }}
  >
    <textarea
      bind:value={sql}
      class="field min-h-30 resize-y rounded-lg font-mono text-sm [tab-size:2]"
      placeholder={PLACEHOLDER}
      {disabled}
      rows={5}
      spellcheck="false"
    ></textarea>
    <div class="mt-2 flex flex-wrap items-center gap-4">
      <button type="submit" class="btn-primary" disabled={disabled || running || !sql.trim()}>
        {running ? "Đang chạy…" : "Thực thi (Ctrl+Enter)"}
      </button>
      {#if execTime !== null}
        <span class="text-sm text-ink-muted">{rows.length} kết quả · {execTime}ms</span>
      {/if}
      {#if db?.lastCost}
        <!-- What this query cost, then what the session has cost so far. -->
        <span class="text-sm text-ink-subtle">
          Truy vấn này: {db.lastCost.requests} yêu cầu · {formatBytes(db.lastCost.bytes)}
          · Phiên: {db.requests} yêu cầu · {formatBytes(db.bytesRead)}
        </span>
      {/if}
    </div>
  </form>

  {#if queryError}
    <p class="notice bg-error-bg text-error-ink">Lỗi: {queryError}</p>
  {/if}

  {#if columns.length > 0}
    <div class="mt-4 overflow-x-auto">
      <table class="w-full border-collapse bg-surface text-sm max-sm:text-xs">
        <thead>
          <tr
            class="[&>th]:sticky [&>th]:top-0 [&>th]:border-b-2 [&>th]:border-line-strong
                   [&>th]:bg-surface-alt [&>th]:px-2 [&>th]:py-2.5 [&>th]:text-left
                   [&>th]:whitespace-nowrap"
          >
            {#each columns as col (col)}
              <th>{col}</th>
            {/each}
          </tr>
        </thead>
        <tbody>
          {#each rows as row, ri (ri)}
            <tr
              class="hover:bg-surface-alt [&>td]:border-b [&>td]:border-line [&>td]:px-2 [&>td]:py-2"
            >
              {#each columns as col (col)}
                <td class="text-center font-medium tabular-nums">
                  {row[col] === null ? "NULL" : String(row[col])}
                </td>
              {/each}
            </tr>
          {/each}
        </tbody>
      </table>
      {#if rows.length >= MAX_ROWS}
        <p class="notice bg-warning-bg text-sm text-warning-ink">
          Hiển thị tối đa {MAX_ROWS} kết quả. Thêm LIMIT để giới hạn.
        </p>
      {/if}
    </div>
  {/if}
</div>
