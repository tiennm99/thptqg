<script>
  import { browser } from "$app/environment";
  import { base, resolve } from "$app/paths";
  import { page } from "$app/state";
  import CustomQuery from "$lib/components/custom-query.svelte";
  import DownloadGate from "$lib/components/download-gate.svelte";
  import ScoreTable from "$lib/components/score-table.svelte";
  import SearchForm from "$lib/components/search-form.svelte";
  import StudentDetail from "$lib/components/student-detail.svelte";
  import { LocalDatabase, formatBytes } from "$lib/database.svelte";
  import { forgetAll } from "$lib/db-cache";
  import { dbOf } from "$lib/datasets";
  import { isExamId } from "$lib/query-mode";
  import { MAX_RESULTS, lookupExamId, searchByName } from "$lib/search";

  let { data } = $props();
  const dataset = $derived(data.dataset);

  let db = $state(null);
  let results = $state(null);
  let searchError = $state(null);
  let searching = $state(false);
  /** How long the last search took, in ms. */
  let searchMs = $state(null);
  let activeTab = $state("search");
  // Owned here, not in SearchForm, so it can be bound to the URL both ways.
  // The query string is unreadable while prerendering — there is no request —
  // so a deep link is picked up on the client only.
  let query = $state(browser ? (page.url.searchParams.get("q") ?? "") : "");

  const busy = $derived(!db?.ready);

  // Created in the browser only: $effect does not run while prerendering. The
  // download itself waits for the visitor to accept it.
  $effect(() => {
    const opened = new LocalDatabase(dbOf(dataset, base), dataset.dbSizeMb * 1024 * 1024);
    db = opened;
    // Consults the cache, and opens a stored copy without asking: consent was
    // given the first time, and reusing it costs nothing.
    void opened.prepare();
    return () => {
      opened.close();
      db = null;
    };
  });

  /** Drop the stored copy, then reload so the gate asks again. */
  async function clearStored() {
    await forgetAll();
    location.reload();
  }

  // Hydrate a ?q= deep link as soon as the database is open.
  let hydrated = false;
  $effect(() => {
    if (!db?.ready || hydrated) return;
    hydrated = true;
    if (query) void search(query);
  });

  // Sync the query to ?q= without adding a history entry, so back still leaves
  // the page and a copied URL still reproduces the search.
  function writeUrlQuery(q) {
    const route = resolve("/[dataset]", { dataset: dataset.id });
    // The native History API rather than SvelteKit's replaceState: that one
    // reaches into the client router's root component, which is undefined
    // until the router has initialized, and then fails inside its own
    // internals (sveltejs/kit#12204). Passing the current history.state back
    // unchanged leaves SvelteKit's history bookkeeping intact, and nothing
    // here reads page.url after the initial deep link.
    history.replaceState(history.state, "", q ? `${route}?q=${encodeURIComponent(q)}` : route);
  }

  async function search(q) {
    const source = db;
    if (!source?.ready) return;
    searchError = null;
    query = q;
    writeUrlQuery(q);

    // A name search scans every row, which is a few hundred milliseconds of
    // blocked main thread. Yielding a frame first lets the spinner paint.
    searching = true;
    searchMs = null;
    const startedAt = performance.now();
    await new Promise(requestAnimationFrame);

    try {
      results = isExamId(q) ? lookupExamId(source, q) : searchByName(source, q);
    } catch (err) {
      searchError = err instanceof Error ? err.message : String(err);
    } finally {
      searchMs = performance.now() - startedAt;
      searching = false;
    }
  }

  function clearSearch() {
    results = null;
    query = "";
    searchMs = null;
    searchError = null;
    writeUrlQuery("");
  }

  // Global shortcuts: Ctrl+Enter submits the SQL query, "/" focuses the search
  // box unless the user is already typing somewhere.
  function onKeydown(event) {
    if (event.ctrlKey && event.key === "Enter" && activeTab === "sql") {
      document.querySelector(".query-form")?.requestSubmit();
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
    <!-- The gate carries the load state for a download. A copy already on the
         device opens without one, so it reports itself here instead. -->
    {#if db?.loading && db.fromCache}
      <p class="notice flex items-center justify-center gap-2 bg-surface-alt" aria-live="polite">
        <span
          class="inline-block h-4 w-4 animate-spin rounded-full border-2 border-line
                 border-t-primary"
          aria-hidden="true"
        ></span>
        Đang mở dữ liệu đã lưu trên máy… {Math.round(db.progress * 100)}%
      </p>
    {/if}

    <div class="mx-auto mb-6 flex max-w-[600px] border-b-2 border-line">
      <button
        class="-mb-0.5 flex-1 cursor-pointer border-0 border-b-2 bg-transparent px-4 py-3
               transition-colors hover:text-primary"
        class:border-transparent={activeTab !== "search"}
        class:text-ink-muted={activeTab !== "search"}
        class:border-primary={activeTab === "search"}
        class:text-primary={activeTab === "search"}
        class:font-semibold={activeTab === "search"}
        onclick={() => (activeTab = "search")}
      >
        Tra cứu
      </button>
      <button
        class="-mb-0.5 flex-1 cursor-pointer border-0 border-b-2 bg-transparent px-4 py-3
               transition-colors hover:text-primary"
        class:border-transparent={activeTab !== "sql"}
        class:text-ink-muted={activeTab !== "sql"}
        class:border-primary={activeTab === "sql"}
        class:text-primary={activeTab === "sql"}
        class:font-semibold={activeTab === "sql"}
        onclick={() => (activeTab = "sql")}
      >
        Truy vấn SQL
      </button>
    </div>

    {#if activeTab === "search"}
      <SearchForm
        value={query}
        onSearch={search}
        onClear={clearSearch}
        disabled={busy || searching}
        examples={dataset.examples}
      />

      {#if searching}
        <p class="notice flex items-center justify-center gap-2 bg-surface-alt" aria-live="polite">
          <span
            class="inline-block h-4 w-4 animate-spin rounded-full border-2 border-line
                   border-t-primary"
            aria-hidden="true"
          ></span>
          Đang tra cứu…
        </p>
      {/if}

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

      {#if searchMs !== null && !searching}
        <p class="mt-2 text-center text-xs text-ink-subtle">
          Truy vấn tại chỗ trong {Math.round(searchMs)} ms
        </p>
      {/if}
    {:else}
      <CustomQuery {db} disabled={busy} presets={dataset.presets} />
    {/if}
  </main>

  <footer class="mt-12 border-t border-line pt-4 text-center text-sm text-ink-subtle">
    <p>
      Nguồn:
      <!-- An off-site article URL, so there is no route for resolve() to take. -->
      <!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
      <a href={dataset.source} title={dataset.source} target="_blank" rel="noopener noreferrer"
        >{dataset.sourceName}</a
      >
      · {dataset.rows.toLocaleString("vi-VN")} thí sinh · Dữ liệu chỉ mang tính tham khảo
    </p>
    {#if db?.fromCache}
      <p class="mt-2">
        Cơ sở dữ liệu ({formatBytes(db.expectedBytes)}) đã lưu trên máy để không phải tải lại.
        <button
          type="button"
          class="cursor-pointer border-0 bg-transparent p-0 text-inherit underline"
          onclick={clearStored}
        >
          Xoá khỏi máy
        </button>
      </p>
    {/if}
  </footer>
</div>

{#if db && !db.ready && db.checked && !db.fromCache}
  <DownloadGate {db} onDownload={() => db.open()} />
{/if}
