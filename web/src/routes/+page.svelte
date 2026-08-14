<script>
  import { resolve } from "$app/paths";
  import { DATASETS } from "$lib/datasets";

  // Trailing slash on purpose: each page is prerendered as <id>/index.html, and
  // hitting it without the slash costs a GitHub Pages directory redirect.
  // resolve() still supplies the base path.
  function hrefOf(dataset) {
    return `${resolve("/[dataset]", { dataset: dataset.id })}/`;
  }
</script>

<svelte:head>
  <title>Tra cứu điểm thi THPT Quốc gia</title>
  <meta
    name="description"
    content="Tra cứu điểm thi tốt nghiệp THPT Quốc gia theo số báo danh hoặc họ tên, chạy hoàn toàn trên trình duyệt."
  />
</svelte:head>

<div class="mx-auto my-12 max-w-2xl px-4 leading-relaxed">
  <header>
    <h1 class="mb-2 text-2xl font-semibold">Tra cứu điểm thi THPT Quốc gia</h1>
    <p class="text-[0.95rem] text-ink-muted">
      Tra cứu điểm thi tốt nghiệp THPT Quốc gia theo số báo danh — truy vấn SQL chạy hoàn toàn trên
      trình duyệt (sql.js).
    </p>
  </header>

  <main>
    <ul class="m-0 my-6 grid list-none gap-3 p-0">
      {#each DATASETS as dataset (dataset.id)}
        <li>
          <!-- hrefOf() is resolve() plus the trailing slash; see above. -->
          <!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
          <a href={hrefOf(dataset)}
            class="flex flex-wrap items-baseline gap-x-3 gap-y-1 rounded-[10px] border border-line
                   bg-surface px-4.5 py-3.5 text-inherit no-underline hover:border-line-strong
                   focus-visible:border-line-strong"
          >
            <span class="font-semibold">{dataset.label}</span>
            <span class="text-sm text-ink-muted">{dataset.blurb}</span>
          </a>
        </li>
      {/each}
    </ul>
  </main>

  <footer class="mt-12 border-t border-line pt-4 text-center text-sm text-ink-subtle">
    <p>Dữ liệu chỉ mang tính tham khảo.</p>
  </footer>
</div>
