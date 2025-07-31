# Response to Design Feedback – ShardExecutor ↔ SDK

Overall I agree with every clarification; no major push-back.

* Executor wiring & ownership – 👍
* Functional option helper – 👍 (easy to add now).
* Error strategy for `ErrQueueFull` – **final decision:** surface back-pressure (`ErrQueueFull`) directly from SDK.  Retry/back-off, if desired, will live in MCP/CLI layers.
    * But expose it to callers as **new** SDK error `ErrBackPressure` to avoid leaking shard details; handlers translate to 503/RESOURCE_EXHAUSTED.
* Read-path noop job – tiny-deadline + `Flush` helper sounds good.
* Stub executor – fully agree.
* Additional client-level metrics – agree but will follow in a later PR to keep this patch focused.

