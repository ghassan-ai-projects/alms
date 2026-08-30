# Resulting Production Go File Inventory

This inventory is generated from the final non-test Go file set. Every row has been inspected. `Refactored` identifies files changed by this task; `Inspected unchanged` identifies files retained because no behavior-preserving clean-code change was justified.

| File | Lines | Disposition | Commit |
|---|---:|---|---|
| `cmd/alms/main.go` | 85 | Refactored | `4149328` |
| `cmd/alms/runtime.go` | 33 | Refactored | `4149328` |
| `internal/config/config.go` | 66 | Refactored | `4fc3105`, `1c4a004` |
| `internal/config/config_env.go` | 12 | Refactored | `4fc3105` |
| `internal/config/config_files.go` | 41 | Refactored | `4fc3105` |
| `internal/models/agent.go` | 81 | Inspected unchanged | — |
| `internal/models/errors.go` | 18 | Inspected unchanged | — |
| `internal/models/learning.go` | 134 | Inspected unchanged | — |
| `internal/models/protocol.go` | 28 | Refactored | `429d7e0` |
| `internal/server/agent_heartbeat_tool.go` | 31 | Refactored | `8c83674` |
| `internal/server/agent_list_tool.go` | 40 | Refactored | `8c83674` |
| `internal/server/agent_register_tool.go` | 37 | Refactored | `8c83674` |
| `internal/server/agent_resources.go` | 27 | Refactored | `5f051c7` |
| `internal/server/agent_tools.go` | 15 | Refactored | `8c83674` |
| `internal/server/agent_unregister_tool.go` | 34 | Refactored | `8c83674` |
| `internal/server/agent_update_tool.go` | 44 | Refactored | `8c83674` |
| `internal/server/dashboard.go` | 22 | Inspected unchanged | — |
| `internal/server/enrichment_tools.go` | 44 | Refactored | `8c83674` |
| `internal/server/health_resource.go` | 32 | Refactored | `5f051c7` |
| `internal/server/health_tools.go` | 37 | Refactored | `8c83674` |
| `internal/server/http_server.go` | 32 | Refactored | `55f4755` |
| `internal/server/learning_delete_tool.go` | 31 | Refactored | `8c83674` |
| `internal/server/learning_get_tool.go` | 29 | Refactored | `8c83674` |
| `internal/server/learning_record_tools.go` | 15 | Refactored | `8c83674` |
| `internal/server/learning_resources.go` | 27 | Refactored | `5f051c7` |
| `internal/server/learning_search_tool.go` | 64 | Refactored | `8c83674` |
| `internal/server/learning_store_tool.go` | 66 | Refactored | `8c83674` |
| `internal/server/learning_sync_tools.go` | 79 | Refactored | `8c83674` |
| `internal/server/legacy_tool_registrations.go` | 35 | Refactored | `0130693` |
| `internal/server/middleware.go` | 43 | Refactored | `8b8d835` |
| `internal/server/okf_tools.go` | 44 | Refactored | `8c83674` |
| `internal/server/protocol_management_tools.go` | 65 | Refactored | `8c83674` |
| `internal/server/protocol_resources.go` | 27 | Refactored | `5f051c7` |
| `internal/server/protocol_sync_tools.go` | 53 | Refactored | `8c83674` |
| `internal/server/resources.go` | 34 | Refactored | `5f051c7` |
| `internal/server/server.go` | 64 | Refactored | `55f4755` |
| `internal/server/tool_resources.go` | 42 | Refactored | `5f051c7` |
| `internal/server/tool_results.go` | 17 | Refactored | `8c83674` |
| `internal/server/tools.go` | 19 | Refactored | `8c83674` |
| `internal/service/dedup.go` | 32 | Refactored | `f0ce74d` |
| `internal/service/dedup_exact.go` | 39 | Refactored | `f0ce74d` |
| `internal/service/dedup_near.go` | 128 | Refactored | `f0ce74d` |
| `internal/service/dedup_supersession.go` | 32 | Refactored | `f0ce74d` |
| `internal/service/gc.go` | 108 | Refactored | `9f55d5d` |
| `internal/service/gc_decay.go` | 37 | Refactored | `9f55d5d` |
| `internal/service/gc_sweep.go` | 84 | Refactored | `9f55d5d` |
| `internal/service/interfaces.go` | 46 | Inspected unchanged | — |
| `internal/service/learning.go` | 139 | Refactored | `e8b415a` |
| `internal/service/learning_dedup.go` | 44 | Refactored | `e8b415a` |
| `internal/service/learning_enrichment.go` | 39 | Refactored | `e8b415a` |
| `internal/service/okf.go` | 104 | Refactored | `eaec891` |
| `internal/service/okf_helpers.go` | 82 | Refactored | `eaec891` |
| `internal/service/okf_index.go` | 51 | Refactored | `eaec891` |
| `internal/service/okf_learning_file.go` | 91 | Refactored | `eaec891` |
| `internal/service/okf_options.go` | 22 | Refactored | `eaec891` |
| `internal/service/protocol_service.go` | 29 | Refactored | `e8b415a` |
| `internal/service/protocol_sync.go` | 45 | Refactored | `4d3df2f` |
| `internal/service/registry.go` | 99 | Inspected unchanged | — |
| `internal/service/scoring.go` | 25 | Refactored | `cd04c6d` |
| `internal/service/scoring_adjustments.go` | 87 | Refactored | `cd04c6d` |
| `internal/service/scoring_decay.go` | 91 | Refactored | `cd04c6d` |
| `internal/service/storemock/agent_store.go` | 163 | Refactored | `49ad345` |
| `internal/service/storemock/helpers.go` | 44 | Refactored | `49ad345` |
| `internal/service/storemock/learning_records.go` | 147 | Refactored | `49ad345` |
| `internal/service/storemock/learning_search.go` | 89 | Refactored | `49ad345` |
| `internal/service/storemock/learning_sync.go` | 97 | Refactored | `49ad345` |
| `internal/service/storemock/protocol_store.go` | 148 | Refactored | `49ad345` |
| `internal/service/sync.go` | 35 | Refactored | `4d3df2f` |
| `internal/service/sync_ack.go` | 57 | Refactored | `4d3df2f` |
| `internal/store/agent_store.go` | 176 | Refactored | `759d8a0` |
| `internal/store/agent_store_encoding.go` | 34 | Refactored | `759d8a0` |
| `internal/store/agent_store_queries.go` | 97 | Refactored | `759d8a0` |
| `internal/store/learning_store.go` | 99 | Refactored | `e353800` |
| `internal/store/learning_store_mutations.go` | 70 | Refactored | `e353800` |
| `internal/store/learning_store_rows.go` | 65 | Refactored | `e353800` |
| `internal/store/learning_store_search.go` | 111 | Refactored | `e353800` |
| `internal/store/learning_store_sync.go` | 142 | Refactored | `e353800` |
| `internal/store/postgres.go` | 37 | Inspected unchanged | — |
| `internal/store/protocol_store.go` | 169 | Refactored | `4d79cf8` |
| `tools.go` | 5 | Refactored | `2a8cd08` |

Verification command used for this inventory:

```text
rg --files -g '*.go' -g '!**/*_test.go'
```
