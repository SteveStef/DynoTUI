# DynamoDB Test TUI — Design Doc (Short)

## Goal
A terminal UI (Go) for browsing and editing **test DynamoDB data only**.  
Connects via AWS credentials or DynamoDB Local. Uses natural language to drive safe DynamoDB operations.

Non-goals: production admin, cross-DB support, large-scale search.

---

## Core Features

- AWS profile / region selector
- List tables and show key schema + indexes
- Browse items using **Query / GetItem** (key-aware)
- View / edit / delete single items
- Seed random test data from table schema hints
- Natural-language command bar → structured execution plan → boto3-style calls

---

## Safety Model

- Default read-only unless account/table is allow-listed
- Block `Scan` by default
- All writes require confirmation
- Hard caps:
  - max pages / items / batch size
  - timeouts on reads

---

## Architecture

**TUI (Bubble Tea)**  
↳ **Planner (LLM)** → JSON Plan  
↳ **Validator** → enforces keys, limits, environment  
↳ **Executor** → DynamoDB SDK calls

---

## “Schema” Handling

- Hard schema: from `DescribeTable` (PK, SK, GSIs)
- Soft schema: inferred from sampled items or optional user schema file
- Used only for data generation and validation

---

## Plan Format (LLM Output)

Example — seed 3 random items:

```json
{
  "op": "put_items",
  "table": "Users",
  "count": 3,
  "strategy": "generate_from_schema",
  "generators": {
    "user_id": { "gen": "uuid", "prefix": "test_" },
    "email": { "gen": "email" },
    "status": { "gen": "one_of", "values": ["active","inactive","pending"] }
  }
}
```

```json
{
  "op": "query",
  "table": "Orders",
  "key_condition": {
    "pk": { "name": "pk", "op": "eq", "value": "USER#123" }
  },
  "limit": 50
}
```

| Plan op            | DynamoDB call            |
| ------------------ | ------------------------ |
| `get_item`         | `GetItem`                |
| `query`            | `Query`                  |
| `count`            | `Query` + `Select=COUNT` |
| `put_items`        | `BatchWriteItem`         |
| `update_item`      | `UpdateItem`             |
| `scan` (test only) | `Scan`                   |


### Tech Stack
- Go
- Bubble Tea (TUI)
- AWS SDK for Go v2 (DynamoDB)
- External LLM API (JSON only output)

### MVP

1. Connect, list tables, describe schema
2. Query by PK, view item
3. Edit / delete single item
4. Seed random test data
5. NL command → plan → execute

