<script>
  import { formatBytes } from "$lib/database.svelte";

  /**
   * The gate a visitor passes before the dataset page works.
   *
   * The whole database is downloaded and queried in the browser, so there is
   * nothing to show until it arrives. Saying so plainly — with both numbers
   * that matter, the transfer and the memory — is fairer than a spinner that
   * silently spends 30 MB of someone's mobile data.
   *
   * It has no dismiss: without the database the page has no answers to give.
   */
  let { db, transferBytes = null, onDownload } = $props();

  const percent = $derived(Math.round(db.progress * 100));
</script>

<div
  class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4"
  role="dialog"
  aria-modal="true"
  aria-labelledby="gate-title"
>
  <div class="w-full max-w-[480px] rounded-xl bg-surface p-6 shadow-xl">
    <h2 id="gate-title" class="mb-2 text-xl font-semibold">Tải cơ sở dữ liệu</h2>

    <p class="mb-4 text-[0.95rem] text-ink-muted">
      Trang này tra cứu hoàn toàn trong trình duyệt, không có máy chủ. Bạn cần tải toàn bộ cơ sở dữ
      liệu về máy một lần để bắt đầu.
    </p>

    <dl class="mb-5 grid grid-cols-2 gap-3 text-sm">
      <div class="rounded-lg bg-surface-alt p-3">
        <dt class="text-ink-muted">Dung lượng tải</dt>
        <dd class="text-lg font-semibold">
          {transferBytes ? formatBytes(transferBytes) : "~31 MB"}
        </dd>
      </div>
      <div class="rounded-lg bg-surface-alt p-3">
        <dt class="text-ink-muted">Bộ nhớ (RAM) cần</dt>
        <dd class="text-lg font-semibold">{formatBytes(db.expectedBytes)}</dd>
      </div>
    </dl>

    <p class="mb-5 text-sm text-ink-subtle">
      Dữ liệu được giữ trong bộ nhớ của tab và sẽ mất khi bạn đóng trang, nên lần truy cập sau cần
      tải lại. Không nên dùng trên thiết bị có ít bộ nhớ.
    </p>

    {#if db.error}
      <p class="notice mb-4 bg-error-bg text-error-ink">Tải thất bại: {db.error}</p>
    {/if}

    {#if db.loading}
      <div class="mb-2 h-2 w-full overflow-hidden rounded-full bg-surface-alt">
        <div class="h-full bg-primary transition-[width]" style="width: {percent}%"></div>
      </div>
      <p class="text-center text-sm text-ink-muted" aria-live="polite">
        Đang tải… {formatBytes(db.received)} / {formatBytes(db.expectedBytes)} ({percent}%)
      </p>
    {:else}
      <button type="button" class="btn-primary w-full" onclick={onDownload}>
        {db.error ? "Thử lại" : "Tải xuống và bắt đầu"}
      </button>
    {/if}
  </div>
</div>
