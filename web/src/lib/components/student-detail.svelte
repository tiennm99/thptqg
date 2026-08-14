<script>
  import { computeBlocks, scoreTier } from "$lib/admission-blocks";
  import { SUBJECTS } from "$lib/subjects";

  let { student } = $props();

  // Visible legend, so a user does not have to hover tiles to decode the
  // colours. The ranges must stay in step with scoreTier() in
  // lib/admission-blocks.ts.
  const TIER_LEGEND = [
    { key: "common", symbol: "·", range: "≤ 1", label: "Điểm liệt" },
    { key: "uncommon", symbol: "○", range: "< 5", label: "Chưa đạt" },
    { key: "rare", symbol: "◆", range: "5-6.5", label: "Trung bình" },
    { key: "epic", symbol: "★", range: "6.5-8", label: "Khá" },
    { key: "legendary", symbol: "✦", range: "8-9", label: "Giỏi" },
    { key: "prismatic", symbol: "❖", range: "9-10", label: "Xuất sắc" },
  ];

  let copied = $state(null);

  const blocks = $derived(computeBlocks(student));
  const subjects = $derived(
    SUBJECTS.filter((s) => student[s.key] !== null && student[s.key] !== undefined).map((s) => ({
      key: s.key,
      label: s.label,
      score: student[s.key],
    })),
  );

  function fmt(n) {
    return n === null || n === undefined ? "—" : Number(n).toFixed(2);
  }

  function flash(kind) {
    copied = kind;
    setTimeout(() => (copied = null), 1500);
  }

  function copySbd() {
    navigator.clipboard.writeText(student.so_bao_danh).then(() => flash("sbd"));
  }

  // Formatted summary suitable for pasting into Zalo / Messenger:
  //   Nguyễn Văn A (SBD 49008235)
  //   Toán 8.50 · Ngữ văn 7.25 · ...
  //   Khối A00 22.75
  //   https://...?q=49008235
  function shareSummary() {
    const lines = [
      `${student.ho_ten} (SBD ${student.so_bao_danh})`,
      subjects.map((s) => `${s.label} ${fmt(s.score)}`).join(" · "),
    ];
    if (blocks.length > 0) {
      lines.push(
        blocks
          .slice(0, 3)
          .map((b) => `Khối ${b.code} ${b.total.toFixed(2)}`)
          .join(" · "),
      );
    }
    lines.push(`${window.location.origin}${window.location.pathname}?q=${student.so_bao_danh}`);
    const text = lines.join("\n");

    if (navigator.share) {
      navigator.share({ text }).catch(() => {});
      return;
    }
    navigator.clipboard.writeText(text).then(() => flash("share"));
  }
</script>

<article
  class="mx-auto mt-6 max-w-[900px] rounded-[14px] border border-line bg-surface p-7 shadow-card
         max-sm:p-4"
  aria-label="Chi tiết điểm thí sinh"
>
  <header
    class="mb-5 flex items-start justify-between gap-4 border-b border-line pb-4 max-sm:flex-col"
  >
    <div>
      <h2 class="mb-2 text-2xl tracking-wide max-sm:text-xl">{student.ho_ten}</h2>
      <dl class="m-0 flex flex-wrap gap-8">
        <div class="flex flex-col gap-0.5">
          <dt class="text-xs tracking-widest text-ink-subtle uppercase">SBD</dt>
          <dd class="m-0 flex items-center gap-2">
            <span class="font-mono tabular-nums">{student.so_bao_danh}</span>
            <button
              type="button"
              class="btn-chip text-ink-muted"
              onclick={copySbd}
              aria-label="Sao chép số báo danh"
            >
              {copied === "sbd" ? "✓ Đã chép" : "Chép"}
            </button>
          </dd>
        </div>
        {#if student.ngay_sinh}
          <div class="flex flex-col gap-0.5">
            <dt class="text-xs tracking-widest text-ink-subtle uppercase">Ngày sinh</dt>
            <dd class="m-0 font-mono tabular-nums">{student.ngay_sinh}</dd>
          </div>
        {/if}
        <!-- Only the 2016 dataset carries these; NULL elsewhere. -->
        {#if student.ten_cum_thi}
          <div class="flex flex-col gap-0.5">
            <dt class="text-xs tracking-widest text-ink-subtle uppercase">Cụm thi</dt>
            <dd class="m-0">{student.ten_cum_thi}</dd>
          </div>
        {/if}
        {#if student.gioi_tinh}
          <div class="flex flex-col gap-0.5">
            <dt class="text-xs tracking-widest text-ink-subtle uppercase">Giới tính</dt>
            <dd class="m-0">{student.gioi_tinh}</dd>
          </div>
        {/if}
      </dl>
    </div>
    <div class="shrink-0">
      <button type="button" class="btn-primary" onclick={shareSummary} aria-label="Chia sẻ bảng điểm">
        {copied === "share" ? "✓ Đã chép" : "Chia sẻ"}
      </button>
    </div>
  </header>

  <section aria-labelledby="subjects-heading">
    <h3
      id="subjects-heading"
      class="mb-3 flex flex-wrap items-center justify-between gap-2 text-[0.95rem] font-semibold"
    >
      Điểm môn thi
      <ul class="m-0 ml-auto flex flex-wrap gap-1.5 p-0 text-xs font-normal" aria-label="Chú thích mức điểm">
        {#each TIER_LEGEND as t (t.key)}
          <li
            class="tier-surface tier-{t.key} inline-flex items-center gap-1 rounded-full border
                   px-2 py-0.5"
          >
            <span aria-hidden="true">{t.symbol}</span>
            <span class="font-semibold tabular-nums">{t.range}</span>
            <span>{t.label}</span>
          </li>
        {/each}
      </ul>
    </h3>

    <ul
      class="m-0 mb-6 grid grid-cols-[repeat(auto-fill,minmax(160px,1fr))] gap-2.5 p-0
             max-sm:grid-cols-[repeat(auto-fill,minmax(130px,1fr))]"
    >
      {#each subjects as s (s.key)}
        {@const tier = scoreTier(s.score)}
        <li
          class="tier-surface tier-{tier?.key} flex flex-col gap-1 rounded-[10px] border px-3.5 py-3"
          aria-label="{s.label}: {fmt(s.score)}, {tier?.label}"
        >
          <span class="text-xs tracking-wide">{s.label}</span>
          <span class="font-mono text-2xl leading-tight font-bold tabular-nums">{fmt(s.score)}</span>
          <span class="flex items-center gap-1 text-[0.78rem]">
            <span class="text-sm font-bold" aria-hidden="true">{tier?.symbol}</span>
            {tier?.label}
          </span>
        </li>
      {/each}
    </ul>
  </section>

  {#if blocks.length > 0}
    <section aria-labelledby="blocks-heading">
      <h3
        id="blocks-heading"
        class="mb-3 flex flex-wrap items-center justify-between gap-2 text-[0.95rem] font-semibold"
      >
        Tổng điểm khối thi
        <span class="text-xs font-normal text-ink-subtle">
          (Chỉ tính những khối thí sinh có đủ 3 môn)
        </span>
      </h3>
      <ul class="m-0 flex list-none flex-col gap-1.5 p-0">
        {#each blocks as b (b.code)}
          <li
            class="grid grid-cols-[44px_1fr_auto_auto] items-center gap-3 rounded-[10px] border
                   border-line bg-surface-alt px-3.5 py-2.5
                   max-sm:grid-cols-[36px_1fr_auto] max-sm:gap-2"
          >
            <span
              class="inline-flex w-9 items-center justify-center rounded-md bg-primary px-0 py-0.5
                     text-sm font-bold tracking-wide text-primary-ink"
            >
              {b.code}
            </span>
            <span class="text-sm text-ink-muted">{b.label}</span>
            <span class="font-mono text-xs text-ink-subtle tabular-nums max-sm:hidden">
              {b.parts.map((p) => fmt(p.score)).join(" + ")}
            </span>
            <span class="min-w-14 text-right font-mono text-lg font-bold tabular-nums">
              {b.total.toFixed(2)}
            </span>
          </li>
        {/each}
      </ul>
    </section>
  {/if}
</article>
