# Unsupported Query Types in DynoTUI

This document outlines the specific types of DynamoDB PartiQL queries and operations that are currently **NOT supported** by DynoTUI, due to architectural limitations (PartiQL constraints or the "Fetch-then-Mutate" pattern).

## 1. UPDATE Operations

### ❌ Dynamic Value Generation
You cannot request updates where the value must be generated dynamically for each item by the client.
*   **Examples:**
    *   "Set a random password for every user."
    *   "Set `updated_at` to the current timestamp." (Unless hardcoded as a static string).
    *   "Generate a new UUID for each item."

### ❌ Data Transformations & Math
You cannot perform operations that modify existing data using mathematical or string functions that PartiQL does not support.
*   **Examples:**
    *   "Double the price of all items." (`SET price = price * 2` is not supported in PartiQL).
    *   "Append ' (Legacy)' to all usernames." (String concatenation is not supported).
    *   "Convert the `age` attribute from Number to String."

### ❌ Conditional Logic (If/Else)
You cannot apply different update logic to items based on their values within a single request.
*   **Examples:**
    *   "If age > 18 set status 'adult', else set status 'minor'."

## 2. SELECT Operations (Analytics)

### ❌ Aggregations
DynamoDB PartiQL is not an analytics engine.
*   **Examples:**
    *   "Count how many users are active." (`COUNT(*)`)
    *   "What is the average age?" (`AVG(age)`)
    *   "Sum the total sales." (`SUM(amount)`)

### ❌ Grouping
*   **Examples:**
    *   "Group users by city." (`GROUP BY`)

### ❌ Multi-Table Operations
*   **Examples:**
    *   "Find users who have orders." (`JOIN`)
    *   "Show items from Table A and Table B." (`UNION`)

## 3. INSERT Operations

### ❌ Templated Bulk Inserts
You cannot use the "Bulk Action" planner for creating new items. Inserts must be explicit, independent SQL statements.
*   **Examples:**
    *   "Add 50 users with random names." (The AI cannot generate the random data client-side and pipe it in).

## 4. DDL (Schema Management)

### ❌ Table & Index Operations
DynoTUI is a Data Manipulation (DML) tool, not a Schema Management tool.
*   **Examples:**
    *   `CREATE TABLE`
    *   `DROP TABLE`
    *   `ALTER TABLE`
    *   Creating or Deleting Global Secondary Indexes (GSIs).

## 5. Transactions

### ❌ ACID Transactions
Bulk operations are executed as **Batches**, not **Transactions**.
*   If you request to delete 100 items and the operation fails at item 50, the first 49 items remain deleted. There is no automatic rollback.
