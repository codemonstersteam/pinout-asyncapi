# ADR-001 — a factory and its product-method are two tree nodes, not one

**Status:** accepted · **Date:** 2026-07-26 · **Change:** `001-pipe-arity-coverage-gaps` (lane `patch`)

## Context

Reshaping `FoldReport` and `NewConsumedContract` into bind-then-chain pairs
(`BuildReporter`/`Reporter.Fold`, `BuildContractParser`/`ContractParser.Parse`) forced a choice the
change-delta could not make (`changes/001-pipe-arity-coverage-gaps/change-delta.md` §5.1): does a
factory/product-method pair occupy **one** row in `module-tree.md` §3 or **two**? The tree already
shipped two such pairs (`BuildSpecLoader`/`FileSpecLoader.Load`,
`BuildReportWriter`/`ReportWriter.Write`) and splits both, each factory carrying its own
`contracts.md` §4 unit-test row. The BRD (N7/D1) assumed the opposite — a rename in place, one node
each, §4's total pinned at 49.

## Decision

**Two nodes**, per the tree's own shipped convention. The tree goes 19 → **21** nodes and
`contracts.md` §4's total goes 49 → **51** (two branchless factory rows at N=1 each). The BRD's
pinned **49 is superseded**. The deciding reason is Parnas, not symmetry: the factory and the method
hide *different*, independently changeable decisions — `BuildReporter` hides where time comes from,
`Reporter.Fold` hides the canon-`1.1` report shape; merging them would put two secrets behind one
interface.

## Consequences

The "head, I/O modules and adapters are not unit-tested" rule stays verbatim (no adapter row, no head
row, no I/O row — BRD C0/Q2); the two added rows are *logic* factories, treated exactly as
`BuildSpecLoader` (N=2) and `BuildReportWriter` (N=2) already are. `module-tree.md` §3's "sanctioned
exceptions by arity" list becomes empty. Downstream artifacts keyed to the old numbering read the
renumbering map in `module-tree.md` §3 ("Node numbering history").
