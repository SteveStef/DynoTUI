# Plan: Automated Bulk Mutations for DynoTUI

## Context
DynamoDB PartiQL does not support bulk `UPDATE` or `DELETE` operations on non-key attributes. It requires the full Primary Key (Partition Key and Sort Key) for every mutation.

## Objective
Implement a "Fetch-then-Mutate" pattern that allows users to perform natural language bulk actions (e.g., "Delete all items where status is 'expired'") seamlessly.

## Proposed Workflow

### 1. Prompt Enhancement (`bedrock.go`)
Update the LLM prompt to handle "Ambiguous Mutations":
- **Constraint:** If a user requests an `UPDATE` or `DELETE` without providing specific Primary Keys (both PK and SK if applicable), or uses non-key attributes in the `WHERE` clause, the LLM should not attempt a direct mutation.
- **Action:** The LLM should instead generate a `SELECT *` query targeting those specific items.
- **Metadata:** Prepend a specific tag or comment to the query, e.g., `-- BULK_DISCOVERY_FOR: DELETE`.

### 2. TUI Logic Update (`update.go`)
Modify the `sqlGeneratedMsg` and execution handlers:
- **Detection:** If the generated SQL is a `SELECT` but carries the `-- BULK_DISCOVERY` metadata:
    - Display a special confirmation: "This action requires two steps. First, we will find the matching items. Continue?"
- **Execution (Step 1):** Execute the `SELECT` query and store the results in `m.items`.
- **Automatic Step 2:** After the items are loaded, display a "Bulk Action" overlay:
    - "Found X items matching your criteria. Apply [DELETE/UPDATE] to all?"
- **Batching:** If the user confirms, generate individual PartiQL statements for each item. 
    - **CRITICAL:** Every statement MUST include the equality match for both the Partition Key AND the Sort Key (if the table has one). Failure to include the SK on a table that has one will result in a ValidationException.
- **Batch Execution:** Execute them using `BatchExecuteStatement` in chunks of 25.

### 3. Implementation Details

#### Step A: Multi-Statement Execution
- Refactor `BatchSqlQuery` in `aws.go` to handle larger batches by chunking requests into groups of 25 (DynamoDB's limit).

#### Step B: Progress UI
- Add a progress bar or counter to `view.go` to show the status of bulk operations (e.g., "Deleting 45/100...").

## Tasks for Execution
- [ ] Update `InvokeBedrock` prompt in `bedrock.go` with "Discovery Query" logic.
- [ ] Update `model` struct in `model.go` to track `bulkActionPending` state.
- [ ] Implement `ApplyBulkActionCmd` in `commands.go` to iterate through `m.items` and generate/execute batch mutations.
- [ ] Add `viewBulkConfirmation` to `view.go`.
