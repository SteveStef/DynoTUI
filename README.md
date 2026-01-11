# DynoTUI

DynoTUI is a terminal-based user interface (TUI) for exploring and managing AWS DynamoDB tables. It leverages **AWS Bedrock (Amazon Nova Lite)** to allow you to interact with your data using natural language, translating your requests into DynamoDB PartiQL statements.

## Features

- **Table Explorer**: View all tables in your region with schema details (PK, SK, Indexes, Item Count).
- **Data Browser**: 
  - Scan tables with pagination support (load 1000 items at a time).
  - View item details in a dedicated JSON inspector.
- **Natural Language Querying**: 
  - Press `/` and ask questions like *"Find users with status ACTIVE"* or *"Insert a new item with id 123"*.
  - Uses **Amazon Nova Lite** via AWS Bedrock to generate optimized PartiQL queries.
  - **Safety First**: 
    - Warns you if a generated query will cause a **Full Table Scan**.
    - Requires confirmation before executing generated SQL.
    - Automatic table refresh after mutations (Insert/Update/Delete).
- **Item Management**:
  - **Edit**: Modify items using your default text editor (`EDITOR` env var).
  - **Add**: Create new JSON items from scratch.
  - **Delete**: Remove items with confirmation.

## Prerequisites

1.  **Go 1.21+** installed.
2.  **AWS Credentials** configured in your environment (e.g., `~/.aws/credentials` or `AWS_ACCESS_KEY_ID`/`AWS_SECRET_ACCESS_KEY`).
3.  **AWS Bedrock Access**:
    - Your AWS account must have access to the **Amazon Nova Lite** model (`amazon.nova-lite-v1:0`) in the `us-east-1` region (currently hardcoded, configurable in code).
    - Ensure your IAM role has `bedrock:InvokeModel` permissions.

## Installation & Running

```bash
# Clone the repository
git clone https://github.com/stevestef/dynotui.git
cd dynotui

# Run directly
go run .

# Or build binary
go build -o dynotui .
./dynotui
```

## Key Bindings

| Key | Action |
| --- | --- |
| `↑` / `k` | Move Up |
| `↓` / `j` | Move Down |
| `Enter` | Select Table / View Item JSON / Execute Command |
| `Esc` / `q` | Go Back / Cancel |
| `/` | **Open Command Bar (AI Query)** |
| `p` | Load Next Page (Pagination) |
| `e` | Edit selected item |
| `a` | Add new item |
| `d` | Delete selected item |
| `?` | Toggle Help |
| `Ctrl+c` | Quit |

## Natural Language Querying

The core feature of DynoTUI is the ability to write natural language queries.

1.  Select a table.
2.  Press `/`.
3.  Type your request. Examples:
    *   *Query*: "Show me the item with PK 'user_123'"
    *   *Filter*: "Find all items where age is greater than 25" (Will warn about Scan if 'age' is not an index)
    *   *Insert*: "Add a new user with id 555 and name 'Alice'"
    *   *Batch*: "Add two items, one with id 1 and another with id 2"
4.  Review the generated SQL.
5.  Press `Enter` to execute.

## Architecture

*   **Frontend**: [Bubble Tea](https://github.com/charmbracelet/bubbletea) (Go TUI framework).
*   **Backend**: AWS SDK for Go v2.
*   **AI**: Amazon Nova Lite via AWS Bedrock Runtime.

