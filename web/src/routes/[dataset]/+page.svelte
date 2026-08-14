<script lang="ts">
  import { browser } from "$app/environment";
  import { replaceState } from "$app/navigation";
  import { base, resolve } from "$app/paths";
  import { page } from "$app/state";
  import CustomQuery from "$lib/components/custom-query.svelte";
  import ScoreTable from "$lib/components/score-table.svelte";
  import SearchForm from "$lib/components/search-form.svelte";
  import StudentDetail from "$lib/components/student-detail.svelte";
  import { dbOf } from "$lib/datasets";
  import { isExamId, normaliseExamId } from "$lib/query-mode";
  import { SqliteSource, queryRows } from "$lib/sqlite.svelte";
  import { isAsciiOnly, toAscii } from "$lib/to-ascii";
  import type { Student } from "$lib/types";

  const MAX_RESULTS = 100;

  let { data } = $props();
  const dataset = $derived(data.dataset);

  let source = $state<SqliteSource | null>(null);
  let results = $state<Student[] | null>(null);
  let searchError = $state<string | null>(null);
  let activeTab = $state<"search" | "sql">("search");
  let totalCount = $state<number | null>(null);
  // Owned here, not in SearchForm, so it can be bound to the URL both ways.
  // The query string is unreadable while prerendering — there is no request —
  // so a deep link is picked up on the client only.
  let query = $state(browser ? (page.url.searchParams.get("q") ?? "") : "");

  const db = $derived(source?.db ?? null);
  const loading = $derived(source?.loading ?? true);
  const loadError = $derived(source?.error ?? null);
  const progress = $derived(source?.progress ?? 0);
  const busy = $derived(loading || !!loadError);

  // The database is per dataset, and only ever fetched in the browser: $effect
  // does not run while prerendering.
  $effect(() => {
    const opened = new SqliteSource(dbOf(dataset, base));
    source = opened;
    return () => {
      opened.close();
      source = null;
      results = null;
      totalCount = null;
    };
  });

  // Candidate count for the footer. One-shot per database: it cannot change
  // without a new one.
  $effect(() => {
    if (!db) return;
    const [row] = queryRows<{ c: number }>(db, "SELECT COUNT(*) AS c FROM student");
    totalCount = row?.c ?? null;
  });

  // Hydrate a ?q= deep link as soon as the database is ready.
  let hydrated = false;
  $effect(() => {
    if (!db || hydrated) return;
    hydrated = true;
    if (query) search(query);
  });

  // Sync the query to ?q= without adding a history entry, so back still leaves
  // the page and a copied URL still reproduces the search.
  function writeUrlQuery(q: string) {
    const route = resolve("/[dataset]", { dataset: dataset.id });
    // The target IS resolve()'d; the lint rule cannot see through the template
    // literal that appends the query string.
    // eslint-disable-next-line svelte/no-navigation-without-resolve
    replaceState(q ? `${route}?q=${encodeURIComponent(q)}` : route, page.state);
  }

  function search(q: string) {
    if (!db) return;
    searchError = null;
    query = q;
    writeUrlQuery(q);

    try {
      if (isExamId(q)) {
        // Letter-prefixed 2016 IDs are stored upper-case; digits are unaffected.
        results = queryRows<Student>(
          db,
          "SELECT * FROM student WHERE so_bao_danh = $q LIMIT $limit",
          { $q: normaliseExamId(q), $limit: MAX_RESULTS },
        );
      } else if (isAsciiOnly(q)) {
        results = queryRows<Student>(
          db,
          "SELECT * FROM student WHERE ho_ten_ascii LIKE $q LIMIT $limit",
          { $q: `%${toAscii(q)}%`, $limit: MAX_RESULTS },
        );
      } else {
        results = queryRows<Student>(
          db,
          `SELECT * FROM student
           WHERE ho_ten LIKE $q OR ho_ten_ascii LIKE $qn
           LIMIT $limit`,
          { $q: `%${q}%`, $qn: `%${toAscii(q)}%`, $limit: MAX_RESULTS },
        );
      }
    } catch (err) {
      searchError = err instanceof Error ? err.message : String(err);
    }
  }

  function clearSearch() {
    results = null;
    query = "";
    writeUrlQuery("");
  }

  // Global shortcuts: Ctrl+Enter submits the SQL query, "/" focuses the search
  // box unless the user is already typing somewhere.
  function onKeydown(event: KeyboardEvent) {
    if (event.ctrlKey && event.key === "Enter" && activeTab === "sql") {
      document.querySelector<HTMLFormElement>(".query-form")?.requestSubmit();
      return;
    }
    if (
      event.key === "/" &&
      activeTab === "search" &&
      !(event.target instanceof HTMLInputElement) &&
      !(event.target instanceof HTMLTextAreaElement)
    ) {
      event.preventDefault();
      document.getElementById("search-input")?.focus();
    }
  }
</script>

<svelte:head>
  <title>{dataset.title}</title>
  <meta name="description" content={dataset.subtitle} />
</svelte:head>

<svelte:window onkeydown={onKeydown} />

<div class="mx-auto max-w-[1200px] px-4 py-8">
  <header class="mb-8 text-center">
    <h1 class="mb-1 text-3xl font-semibold max-sm:text-xl">{dataset.title}</h1>
    <p class="text-[0.95rem] text-ink-muted">{dataset.subtitle}</p>
    <p class="mt-1.5 text-sm">
      <a class="text-ink-muted no-underline hover:underline" href={resolve("/")}>
        ← Tất cả các kỳ thi
      </a>
    </p>
  </header>

  <main>
    {#if loading}
      <div class="my-4 text-center text-ink-muted">
        <p>
          Đang tải cơ sở dữ liệu ~{dataset.dbSizeMb} MB{progress > 0 ? ` · ${progress}%` : ""}
        </p>
        <div class="mx-auto my-2 h-2 w-[300px] max-w-full overflow-hidden rounded bg-surface-alt">
          <div class="h-full rounded bg-primary transition-[width]" style="width: {progress}%"></div>
        </div>
        <p class="mx-auto mt-2 max-w-[480px] text-sm text-ink-subtle">
          Lần đầu có thể mất 10-30 giây. Sau đó trình duyệt sẽ lưu cache và mở nhanh hơn.
        </p>
      </div>
    {/if}

    {#if loadError}
      <p class="notice bg-error-bg text-error-ink">Lỗi: {loadError}</p>
    {/if}

    <div class="mx-auto mb-6 flex max-w-[600px] border-b-2 border-line">
      {#each [{ id: "search", label: "Tra cứu" }, { id: "sql", label: "Truy vấn SQL" }] as const as tab (tab.id)}
        <button
          class="-mb-0.5 flex-1 cursor-pointer border-0 border-b-2 bg-transparent px-4 py-3
                 transition-colors hover:text-primary"
          class:border-transparent={activeTab !== tab.id}
          class:text-ink-muted={activeTab !== tab.id}
          class:border-primary={activeTab === tab.id}
          class:text-primary={activeTab === tab.id}
          class:font-semibold={activeTab === tab.id}
          onclick={() => (activeTab = tab.id)}
        >
          {tab.label}
        </button>
      {/each}
    </div>

    {#if activeTab === "search"}
      <SearchForm
        value={query}
        onSearch={search}
        onClear={clearSearch}
        disabled={busy}
        examples={dataset.examples}
      />

      {#if searchError}
        <p class="notice bg-error-bg text-error-ink">Lỗi truy vấn: {searchError}</p>
      {/if}

      {#if results && results.length === 1}
        <StudentDetail student={results[0]} />
      {:else}
        <ScoreTable {results} />
      {/if}

      {#if results && results.length >= MAX_RESULTS}
        <p class="notice bg-warning-bg text-sm text-warning-ink">
          Hiển thị tối đa {MAX_RESULTS} kết quả. Vui lòng tìm kiếm cụ thể hơn.
        </p>
      {/if}
    {:else}
      <CustomQuery {db} disabled={busy} presets={dataset.presets} />
    {/if}
  </main>

  <footer
    class="mt-12 border-t border-line pt-4 text-center text-sm break-words text-ink-subtle"
  >
    <p>
      Nguồn:
      <a href={dataset.source} target="_blank" rel="noopener noreferrer">{dataset.source}</a>
      {#if totalCount !== null}
        · {totalCount.toLocaleString("vi-VN")} thí sinh
      {/if}
      · Dữ liệu chỉ mang tính tham khảo
    </p>
  </footer>
</div>
