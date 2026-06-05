# ALMS — Sync Flow & Gap-Safe Ack Algorithm

## Overview

The sync flow is the core value proposition of ALMS. Agents call `learning.sync` to get new learnings since their last acknowledged timestamp, then call `learning.sync_ack` to confirm processing. The gap-safe validation ensures no learnings are lost if an agent crashes mid-batch.

## Sequence Diagram

```
Agent A                          ALMS                          Other Agents
   │                               │                               │
   │  learning.sync(               │                               │
   │    agent_id="A",             │                               │
   │    since_timestamp=           │                               │
   │      "2026-06-04T00:00:00Z") │                               │
   │──────────────────────────────>│                               │
   │                               │  SELECT * FROM learnings     │
   │                               │  WHERE created_at > $1       │
   │                               │    AND NOT is_deleted        │
   │                               │  ORDER BY created_at ASC     │
   │                               │                               │
   │   [LRN-006, LRN-007,         │                               │
   │    LRN-008]                  │                               │
   │<──────────────────────────────│                               │
   │                               │                               │
   │  Agent processes learnings   │                               │
   │  (may crash here)            │                               │
   │                               │                               │
   │  learning.sync_ack(           │                               │
   │    agent_id="A",             │                               │
   │    learning_ids=[            │                               │
   │      "LRN-006",             │                               │
   │      "LRN-007",             │                               │
   │      "LRN-008"])            │                               │
   │──────────────────────────────>│                               │
   │                               │  Validate: no gaps?          │
   │                               │  INSERT INTO                 │
   │                               │    learning_acknowledgements  │
   │                               │  UPDATE agents               │
   │                               │    SET last_sync_ts = $now   │
   │                               │                               │
   │   {ok}                       │                               │
   │<──────────────────────────────│                               │
```

## Gap-Safe Ack Algorithm (Pseudocode)

```
function SyncAck(agentID, learningIDs):
    // 1. Get the agent's current sync state
    agent = AgentStore.Get(agentID)
    
    // 2. Get all learnings the agent SHOULD have received
    expected = LearningStore.Sync(agentID, agent.last_sync_ts, "", nil)
    expectedIDs = expected.map(l => l.learning_id)
    
    // 3. Check for gaps between expected and acknowledged
    if len(expectedIDs) == 0:
        return OK  // nothing to ack
    
    ackSet = set(learningIDs)
    missing = []
    for id in expectedIDs:
        if id not in ackSet:
            missing.append(id)
    
    if len(missing) > 0:
        return ErrGapDetected(missing)
    
    // 4. Check for unknown IDs (agent acked something not in expected)
    unknown = learningIDs.filter(id => id not in set(expectedIDs))
    // Unknown IDs are silently ignored (they may have been deleted)
    
    // 5. Persist acknowledgements
    for id in learningIDs:
        INSERT INTO learning_acknowledgements (agent_id, learning_id)
        VALUES ($1, $2)
        ON CONFLICT DO NOTHING  // idempotent
    
    // 6. Advance agent's sync cursor to newest acked learning's timestamp
    if len(learningIDs) > 0:
        newestID = learningIDs.last()  // IDs are ordered by created_at
        newestLearning = LearningStore.Get(newestID)
        AgentStore.Update(agentID, { last_sync_ts: newestLearning.created_at })
    
    return OK
```

## Crash Recovery Scenario

```
State: Agent A last_sync_ts = "2026-06-04T00:00:00Z"

1. Agent A calls sync(since="2026-06-04")
   → Returns: [LRN-006, LRN-007, LRN-008]

2. Agent A processes LRN-006, LRN-007
   → CRASHES before LRN-008

3. Agent A restarts, calls sync(since="2026-06-04")
   → Returns: [LRN-006, LRN-007, LRN-008]
   (LRN-006 and LRN-007 were never acked → they reappear)

4. Agent A calls sync_ack(["LRN-006", "LRN-007", "LRN-008"])
   → OK

5. Agent A calls sync(since=latest_ack_timestamp)
   → Returns: []  (empty, all caught up)
```
