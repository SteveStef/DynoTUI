# DynoTUI Query Capabilities

This document outlines the queries that are supported by DynoTUI's "Fetch-then-Mutate" engine and those that are explicitly unsupported or refused.

## ✅ Supported Queries (What Works)

These queries are executed safely, either directly as SQL or via a multi-step Plan.

### 1. Direct SQL (Single Item Operations)
*   **Get Item:** "Get the user with id 123"
*   **Insert Item:** "Add a new user with id 55 and name 'Alice'"
*   **Update Item:** "Update user 123, set status to 'active'"
*   **Delete Item:** "Delete the item with id 123"

### 2. Bulk Updates (Static Values)
*   **Update All:** "Set status to 'archived' for all items where age > 50"
*   **Update Field:** "Change category to 'Legacy' for all items"
*   **Copy Field:** "Set backup_email equal to email for all users" (Intra-item reference)

### 3. Bulk Deletes
*   **Conditional Delete:** "Delete all items where status is 'inactive'"
*   **Delete All:** "Delete everything in the table" (Will show a Scan Warning)

### 4. Batch Inserts (Explicit)
*   **Batch Add:** "Add 3 items: {id: 1, name: 'a'}, {id: 2, name: 'b'}, {id: 3, name: 'c'}"
    *   *Note:* The AI must generate the full SQL for each item.

---

## ❌ Unsupported / Refused Queries (What Fails)

These requests will trigger a **"Refusal"** response or a **"Safety Error"**.

### 1. Dynamic Value Generation
The client cannot generate unique data per item.
*   ❌ "Set a random password for everyone."
*   ❌ "Update `updated_at` to the current timestamp."
*   ❌ "Generate a UUID for each new item."

### 2. Aggregations (Analytics)
DynamoDB PartiQL does not support SQL aggregations.
*   ❌ "What is the average age?" (`AVG`)
*   ❌ "Count how many users are active." (`COUNT`)
*   ❌ "Sum the total price." (`SUM`)

### 3. Data Transformations
*   ❌ "Uppercase all usernames." (No `UPPER()` function).
*   ❌ "Set full_name = first + ' ' + last." (No string concatenation).
*   ❌ "Double the points for every user." (No math operations in updates).

### 4. Cross-Table Operations
*   ❌ "Find users who have orders." (`JOIN`)
*   ❌ "Show me items from Table A and Table B." (`UNION`)

### 5. Conditional Logic
*   ❌ "If age > 18 set status 'adult', else set status 'minor'." (Requires branching logic).

### 6. Schema Management (DDL)
*   ❌ "Create a new table."
*   ❌ "Drop this table."
*   ❌ "Add an index on email."
