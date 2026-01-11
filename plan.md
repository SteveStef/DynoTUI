LLMResult Rules
Root Structure
{
  "mode": "sql" | "plan",
  "statements": [ "<PartiQL statement>" ],
  "plan": { ... } | null
}

Mode Rules
mode	Allowed content
sql	statements MUST contain at least one valid PartiQL statement. plan MUST be null.
plan	statements MUST be an empty array []. plan MUST NOT be null.
statements Rules

Used only when mode = "sql".

Must contain only safe, key-bounded PartiQL statements.

Each statement must be independently executable by ExecuteStatement.

No statement in statements may perform a multi-item mutation without full primary key equality.

plan Object Rules
plan Structure
{
  "table": "<table name>",
  "operation": "select" | "scan_then_write",
  "read": { ... },
  "write": null | { ... },
  "safety": { ... }
}

operation Rules
operation	meaning
select	Read-only operation. May involve Query or Scan.
scan_then_write	Multi-item mutation. Requires read + write.
read Rules
{
  "partiql": "<SELECT statement>",
  "requires_scan": true | false,
  "index": "<GSI name>" | null,
  "projection": ["<PK>", "<SK>"] | ["*"]
}


partiql MUST be a SELECT.

projection MUST include the full primary key (PK, and SK if the table has one) unless projection = ["*"].

If the query does not use a partition key or GSI partition key, requires_scan MUST be true.

write Rules

Must be null when operation = "select".

Must NOT be null when operation = "scan_then_write".

{
  "action": "insert" | "update" | "delete",
  "per_item": {
    "partiql_template":
      "Key-bounded PartiQL with {{PK}} and {{SK}} placeholders"
  }
}


partiql_template MUST include {{PK}}.

If the table has a sort key, {{SK}} MUST also be present.

The template MUST uniquely target exactly one item.

safety Rules
{
  "needs_confirmation": true | false,
  "reason": "none" | "full_table_scan" | "multi_item_write"
}


needs_confirmation MUST be true when:

read.requires_scan = true, or

operation = "scan_then_write".

Hard Invariants
Condition	Enforcement
mode = "plan"	statements MUST be empty
mode = "sql"	statements MUST NOT be empty
operation = "scan_then_write"	write MUST NOT be null
Missing PK or SK in mutation	MUST be rejected
Multi-item UPDATE / DELETE without keys	MUST use scan_then_write

These rules guarantee that DynoTUI never executes unsafe DynamoDB operations.
