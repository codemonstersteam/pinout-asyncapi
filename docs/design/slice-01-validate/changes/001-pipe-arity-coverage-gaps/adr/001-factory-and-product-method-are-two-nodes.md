# ADR-001 — a factory and its product-method are two tree nodes, not one

**Status:** accepted · **Date:** 2026-07-26 · **Change:** `001-pipe-arity-coverage-gaps` (lane `patch`)

> **Moved to the canonical package.** The authoritative text lives in
> [`../../adr/001-factory-and-product-method-are-two-nodes.md`](../../adr/001-factory-and-product-method-are-two-nodes.md).
>
> Per `docs/05_REPO_STRUCTURE.md` ("Канон живой, дельта — провенанс") the slice package states the
> current design; this change folder is the record of what moved and why. An ADR is additive — it is
> not undone by a later change — so it belongs in the canon, and keeping a second, drifting copy here
> would give two answers to one question. This pointer preserves the provenance link: the decision was
> taken **in this change**, on the question `change-delta.md` §5.1 could not settle (one node per
> factory/product pair, or two).
>
> Consequence recorded there and folded into the canon: 19 → 21 nodes, 49 → 51 unit rows.
