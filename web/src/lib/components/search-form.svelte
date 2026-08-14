<script>
  import { detectMode } from "$lib/query-mode";

  let { value = "", onSearch, onClear, disabled = false, examples = [] } = $props();

  // A writable derived: it follows the owner's value — which is bound to the
  // URL, so a deep link or the clear button flows in — and typing overrides it
  // until the next external change.
  let query = $derived(value);
  let input = $state(null);

  const detected = $derived(detectMode(query));
  const canSearch = $derived(detected.mode === "sbd" || detected.mode === "name");

  // Searching only on submit, never while typing. A name search seeks on the
  // rarest word the query contains, and a half-typed word is a wide prefix:
  // "bu" covers every Bùi, Bửu and Bưu in the dataset, and each partial
  // keystroke used to pay for that in full over the network.
  function submit(event) {
    event.preventDefault();
    if (!canSearch) return;
    onSearch(query.trim());
  }

  function clear() {
    query = "";
    input?.focus();
    onClear?.();
  }
</script>

<form onsubmit={submit} class="mx-auto mb-6 flex max-w-[640px] flex-col gap-1.5" role="search">
  <label for="search-input" class="text-sm font-semibold tracking-wider text-ink-muted uppercase">
    Tra cứu điểm
  </label>

  <div class="flex gap-2 max-sm:flex-col">
    <div class="relative flex flex-1 items-center">
      <input
        id="search-input"
        bind:this={input}
        bind:value={query}
        type="search"
        inputmode="search"
        class="field pr-9 [&::-webkit-search-cancel-button]:hidden"
        placeholder="Ví dụ: 47006585 hoặc Nguyen Van An"
        {disabled}
        autocomplete="off"
        spellcheck="false"
        aria-describedby="search-hint"
      />
      {#if query}
        <button
          type="button"
          class="absolute right-1.5 flex h-7 w-7 cursor-pointer items-center justify-center
                 rounded-full border-0 bg-transparent text-xl leading-none text-ink-muted
                 transition-colors hover:bg-surface-alt hover:text-ink"
          onclick={clear}
          aria-label="Xoá tìm kiếm"
          tabindex="-1"
        >
          ×
        </button>
      {/if}
    </div>

    <button type="submit" class="btn-primary max-sm:w-full" disabled={disabled || !canSearch}>
      Tra cứu
    </button>
  </div>

  <p
    id="search-hint"
    class="m-0 min-h-[1.2em] text-sm transition-colors"
    class:text-ink-muted={detected.mode !== "sbd-short" && detected.mode !== "name-short"}
    class:text-[color:var(--tier-legendary-accent)]={detected.mode === "sbd-short" ||
      detected.mode === "name-short"}
    aria-live="polite"
  >
    {detected.hint}{#if detected.mode === "empty"} · VD: {#each examples as example, i (example)}<button
          type="button"
          class="btn-chip"
          onclick={() => {
            query = example;
            input?.focus();
          }}>{example}</button
        >{i < examples.length - 1 ? " hoặc " : ""}{/each}{/if}
  </p>
</form>
