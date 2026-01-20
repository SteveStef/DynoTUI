# DynoTUI Project Documentation

## Overview
**DynoTUI** is a terminal user interface (TUI) for managing Amazon DynamoDB tables. It is written in **Go** and utilizes the **Bubble Tea** framework for the UI. A key feature of DynoTUI is its integration with **AWS Bedrock** (specifically the Amazon Nova Lite model), allowing users to query and manipulate their database using natural language.

## Architecture
The application follows **The Elm Architecture** (Model-Update-View) provided by Bubble Tea.

### Core Components

1.  **Main Entry Point (`main.go`)**
    *   Initializes the AWS client (loading config and identity once).
    *   Sets up logging.
    *   Starts the Bubble Tea program with the initial model.

2.  **State Management (`model.go`)**
    *   Defines the `model` struct, which holds the entire application state:
        *   `aws`: The shared AWS client instance.
        *   `tables`: List of DynamoDB tables.
        *   `items`: Current list of items being viewed.
        *   `view`: The current UI screen (Loading, TableList, ItemList, Confirmation, etc.).
        *   `llmResult`: The structured response from the AI.
        *   `input`: Text input model for queries.

3.  **UI Logic (`update.go`)**
    *   Handles all events (Key presses, Window resizes, Custom messages).
    *   Dispatches asynchronous commands (e.g., `scanTable`, `generateSQLCmd`).
    *   Manages state transitions (e.g., from Table List -> Table Items).

4.  **UI Rendering (`view.go`, `styles.go`)**
    *   `view.go`: Renders the current state into a string. It supports multiple views:
        *   **Table List**: Displays available tables with metadata (Item count, Region).
        *   **Item View**: A split-pane view with a list of items on the left and a JSON inspector on the right.
        *   **Confirmations**: Dialogs for potentially dangerous operations (AI plans, Deletes).
    *   `styles.go`: Defines Lipgloss styles for colors, borders, and layout.

5.  **AWS Interaction (`aws.go`)**
    *   **`AWS` Struct**: Wraps `dynamodb.Client` and `bedrockruntime.Client`.
    *   **Methods**:
        *   `ListTablesWithDetails`: Lists tables and describes them to get keys and GSIs.
        *   `ScanTable`: Pages through table items.
        *   `SqlQuery`: Executes PartiQL statements.
        *   `BatchSqlQuery`: Handles batch PartiQL operations with chunking.

6.  **AI Integration (`bedrock.go`)**
    *   **`InvokeBedrock`**: Sends user prompts to AWS Bedrock.
    *   **Prompt Engineering**: Uses a sophisticated system prompt to force the LLM to return a strict JSON schema (`LLMResult`).
    *   **Capabilities**:
        *   `sql` mode: Returns direct PartiQL statements for simple queries.
        *   `plan` mode: Returns a "Fetch-then-Mutate" plan for complex multi-item operations (e.g., "Delete all items older than X").
    *   **Safety**: Explicitly instructs the AI to refuse dangerous or unsupported operations (like random value generation or aggregations).

7.  **Async Commands (`commands.go`)**
    *   Wraps blocking AWS calls into `tea.Cmd` functions that return `tea.Msg`.
    *   Examples: `loadTables`, `saveItemCmd`, `generateSQLCmd`.

## File Structure

```text
.
├── aws.go          # AWS Client wrapper (DynamoDB + Bedrock)
├── bedrock.go      # AI Logic, Prompts, and JSON Schema definitions
├── commands.go     # Bubble Tea Commands (Async tasks)
├── editor.go       # Text editor integration (for editing JSON items)
├── keys.go         # Keybindings definition
├── main.go         # Entry point
├── messages.go     # Bubble Tea Message types
├── model.go        # State definitions (Model struct)
├── styles.go       # UI Styling (Lipgloss)
├── update.go       # Event Loop (Update function)
└── view.go         # UI Rendering (View function)
```

## Key Workflows

### 1. App Startup
`main.go` -> `NewAWS()` -> `initialModel(aws)` -> `Update()` triggers `loadTables` -> `ListTablesWithDetails` fetches metadata.

### 2. Natural Language Query
User types query "/" -> `Enter` -> `generateSQLCmd` calls `InvokeBedrock`.
*   **Bedrock** returns JSON.
*   **Update** parses JSON.
    *   If `mode="sql"`: Shows confirmation -> Executes `SqlQuery`.
    *   If `mode="plan"`: Shows confirmation -> Executes Read -> Shows Bulk Confirmation -> Executes Write (`BatchSqlQuery`).

### 3. Editing Items
User presses `e` on an item -> Opens `$EDITOR` with item JSON -> User saves -> `editorFinishedMsg` -> `PutItem` updates DynamoDB -> UI refreshes.

## Developer Notes
*   **PartiQL**: The app relies heavily on DynamoDB's PartiQL support.
*   **Connection Reuse**: The `AWS` struct is shared to avoid re-establishing connections on every request.
*   **Safety**: The AI prompt is the primary defense against invalid queries. It is tuned to refuse unsupported operations (aggregations, schema changes).
