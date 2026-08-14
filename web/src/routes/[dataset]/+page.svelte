<script>
  import { browser } from "$app/environment";
  import { base, resolve } from "$app/paths";
  import { page } from "$app/state";
  import CustomQuery from "$lib/components/custom-query.svelte";
  import ScoreTable from "$lib/components/score-table.svelte";
  import SearchForm from "$lib/components/search-form.svelte";
  import StudentDetail from "$lib/components/student-detail.svelte";
  import { dbSourceOf } from "$lib/datasets";
  import { isExamId } from "$lib/query-mode";
  import { MAX_RESULTS, lookupExamId, searchByName } from "$lib/search";
  import {
    PLAYGROUND_BUDGET_BYTES,
    RemoteDatabase,
    SEARCH_BUDGET_BYTES,
    formatBytes,
  } from "$lib/sqlite.svelte";

  let { data } = $props();
  const dataset = $derived(data.dataset);

  let db = $state(null);
  let results = $state(null);
  let searchError = $state(null);
  let searching = $state(false);
  let elapsedMs = $state(0);
  /** What the last search cost over the network, for the line under the results. */
  let cost = $state(null);
  let activeTab = $state("search");
  let sqlWarningOpen = $state(false);
  // Raised once the user has accepted that a hand-written query may fetch a lot.
  let budget = $state(SEARCH_BUDGET_BYTES);
  // Owned here, not in SearchForm, so it can be bound to the URL both ways.
  // The query string is unreadable while prerendering — there is no request —
  // so a deep link is picked up on the client only.
  let query = $state(browser ? (page.url.searchParams.get("q") ?? "") : "");

  const opening = $derived(db !== null && !db.ready && db.error === null);
  const loadError = $derived(db?.error ?? null);
  const busy = $derived(!db?.ready);

  // Opened in the browser only: $effect does not run while prerendering. A new
  // budget means a new worker, which costs only the header pages.
  $effect(() => {
    const opened = new RemoteDatabase(dbSourceOf(dataset, base), budget);
    db = opened;
    return () => {
      opened.close();
      db = null;
    };
  });

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

    // Counted across the whole search rather than per query: a name search runs
    // two, one for the word frequencies and one for the rows.
    const before = { requests: source.requests, bytes: source.bytesRead };
    const startedAt = performance.now();
    searching = true;
    cost = null;
    // The worker reads pages with synchronous XHR, so it cannot answer while a
    // query runs and there is no request count to show until it finishes.
    // Elapsed time is the one honest live signal.
    elapsedMs = 0;
    const ticking = setInterval(() => (elapsedMs = performance.now() - startedAt), 100);

    try {
      results = isExamId(q) ? await lookupExamId(source, q) : await searchByName(source, q);
    } catch (err) {
      searchError = err instanceof Error ? err.message : String(err);
    } finally {
      clearInterval(ticking);
      searching = false;
      cost = {
        requests: source.requests - before.requests,
        bytes: source.bytesRead - before.bytes,
        ms: performance.now() - startedAt,
        sessionRequests: source.requests,
        sessionBytes: source.bytesRead,
      };
    }
  }

  function clearSearch() {
    results = null;
    query = "";
    cost = null;
    searchError = null;
    writeUrlQuery("");
  }

  /** "16,9 giây", or "0,4 giây" — one decimal is enough to compare searches. */
  function seconds(ms) {
    return `${(ms / 1000).toLocaleString("vi-VN", { minimumFractionDigits: 1, maximumFractionDigits: 1 })} giây`;
  }

  function openSqlTab() {
    if (budget >= PLAYGROUND_BUDGET_BYTES) {
      activeTab = "sql";
      return;
    }
    sqlWarningOpen = true;
  }

  function acceptSqlWarning() {
    sqlWarningOpen = false;
    // Reopening with the larger budget throws away the current worker, and with
    // it the pages it had cached — a few hundred KB, refetched on demand.
    budget = PLAYGROUND_BUDGET_BYTES;
    activeTab = "sql";
  }

  function declineSqlWarning() {
    sqlWarningOpen = false;
    activeTab = "search";
  }

  // Global shortcuts: Ctrl+Enter submits the SQL query, "/" focuses the search
  // box unless the user is already typing somewhere.
  function onKeydown(event) {
    if (event.key === "Escape" && sqlWarningOpen) {
      declineSqlWarning();
      return;
    }
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
    {#if opening}
      <p class="my-4 text-center text-ink-muted">Đang mở cơ sở dữ liệu…</p>
    {/if}

    {#if loadError}
      <p class="notice bg-error-bg text-error-ink">Lỗi: {loadError}</p>
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
        onclick={openSqlTab}
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
          Đang tra cứu… {seconds(elapsedMs)}
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

      {#if cost && !searching}
        <p class="mt-2 text-center text-xs text-ink-subtle">
          Mạng: {cost.requests.toLocaleString("vi-VN")} yêu cầu · {formatBytes(cost.bytes)} · {seconds(
            cost.ms,
          )} · Cả phiên: {cost.sessionRequests.toLocaleString("vi-VN")} yêu cầu · {formatBytes(
            cost.sessionBytes,
          )}
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
  </footer>
</div>

{#if sqlWarningOpen}
  <!--
    The SQL tab is the one place a user can write a query that reads the whole
    database. Everything else here is a seek; this is not, so it is opt-in.
  -->
  <div
    class="fixed inset-0 z-10 flex items-center justify-center bg-black/50 p-4"
    role="dialog"
    aria-modal="true"
    aria-labelledby="sql-warning-title"
  >
    <div class="max-w-[520px] rounded-xl border border-line bg-surface p-6 shadow-card">
      <h2 id="sql-warning-title" class="mb-3 text-lg font-semibold">Truy vấn SQL tốn dữ liệu mạng</h2>
      <p class="mb-3 text-sm text-ink-muted">
        Tra cứu thường chỉ tải vài trăm KB. Truy vấn SQL tự viết có thể quét toàn bộ bảng và tải tới
        hàng trăm MB — tốn dữ liệu di động và có thể rất chậm.
      </p>
      <p class="mb-5 text-sm text-ink-muted">
        Cơ sở dữ liệu này nặng {dataset.dbSizeMb} MB. Số byte đã tải sẽ hiển thị bên cạnh thời gian
        chạy để bạn theo dõi.
      </p>
      <div class="flex flex-wrap justify-end gap-3">
        <button type="button" class="btn-chip" onclick={declineSqlWarning}>
          Quay lại tra cứu
        </button>
        <button type="button" class="btn-primary" onclick={acceptSqlWarning}>
          Tôi hiểu, tiếp tục
        </button>
      </div>
    </div>
  </div>
{/if}
