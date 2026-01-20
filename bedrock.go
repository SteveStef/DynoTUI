package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)

type NovaRequest struct {
	Messages []NovaMessage `json:"messages"`
	InferenceConfig struct {
		MaxNewTokens int     `json:"max_new_tokens"`
		Temperature  float64 `json:"temperature"`
	} `json:"inferenceConfig"`
}

type NovaMessage struct {
	Role    string        `json:"role"`
	Content []NovaContent `json:"content"`
}

type NovaContent struct {
	Text string `json:"text"`
}

type NovaResponse struct {
	Output struct {
		Message struct {
			Content []NovaContent `json:"content"`
		} `json:"message"`
	} `json:"output"`
}

type LLMResult struct {
	Mode          string     `json:"mode"`
	Reason        string     `json:"reason"`
	Statements    []string   `json:"statements"`
	Plan          *PlanBlock `json:"plan"`
	RefusalReason string     `json:"refusal_reason"`
}

type PlanBlock struct {
	Table     string     `json:"table"`
	Operation string     `json:"operation"`
	Read      ReadBlock  `json:"read"`
	Write     *WriteBlock `json:"write"`
	Safety    SafetyBlock `json:"safety"`
}

type ReadBlock struct {
	Partiql       string   `json:"partiql"`
	RequiresScan  bool     `json:"requires_scan"`
	Index         *string  `json:"index"`
	Projection    []string `json:"projection"`
}

type WriteBlock struct {
	Action  string `json:"action"`
	PerItem struct {
		PartiqlTemplate string `json:"partiql_template"`
	} `json:"per_item"`
}

type SafetyBlock struct {
	NeedsConfirmation bool   `json:"needs_confirmation"`
	Reason            string `json:"reason"`
}

func (a *AWS) InvokeBedrock(ctx context.Context, question string, table Table) (LLMResult, error) {
	log.Printf("InvokeBedrock called with question: '%s' for table: '%s'", question, table.Name)

	// Construct schema description
	schemaDesc := fmt.Sprintf("Table Name: %s\nPartition Key: %s (Type: %s)\n", 
		table.Name, table.PK, table.PKType)
	
	if table.SK != "" {
		schemaDesc += fmt.Sprintf("Sort Key: %s (Type: %s)\n", table.SK, table.SKType)
	}
	
	if len(table.GSIs) > 0 {
		schemaDesc += fmt.Sprintf("Global Secondary Indexes: %v\n", table.GSIs)
	}
	prompt := fmt.Sprintf(`
You are a DynamoDB expert. Your job is to produce a SAFE execution plan for DynamoDB.

SYSTEM CAPABILITIES (CRITICAL CONTEXT)
This tool is a DynamoDB Manager. It supports:
1. SQL MODE: Simple, efficient Single-Item Reads/Writes using PartiQL.
2. PLAN MODE (READ): Complex Scans or Queries that return multiple items.
3. PLAN MODE (WRITE): "Fetch-then-Mutate" operations for multi-item updates.

CRITICAL RULES:
- NEVER invent a write operation if the user did not ask for one.
- If the user asks to "Get", "Find", or "Search", use a Read-Only operation (sql or plan with write: null).
- Only use "Fetch-then-Mutate" (write block) if the user explicitly asks to UPDATE, DELETE, or MODIFY data.

1. READ: It runs a SELECT query to find items.
2. WRITE: It applies a *static* PartiQL template to every item found.
3. INSERTs must be fully specified SQL statements (Mode: sql). You cannot use a Plan to generate new items.
It CANNOT generate unique values client-side during a PLAN. However, for INSERT (Mode: sql), YOU (the AI) can and should generate the random/dummy data yourself.
Therefore, any request for "random" or "unique" values in an UPDATE/DELETE plan is IMPOSSIBLE and MUST be refused.

Return EXACTLY ONE valid JSON object and nothing else (no markdown, no backticks, no explanations).

INPUTS
Schema (includes table name, PK/SK, GSIs):
%s
NOTE: The schema above only lists keys and indexes. The table contains other attributes not listed here. Do not refuse a query just because an attribute is not in this schema.

User request:
%s

OUTPUT JSON SCHEMA
{
  "mode": "sql" | "plan" | "refusal",
  "reason": "Brief explanation of why this mode was chosen",
  "statements": ["<PartiQL>"],
  "refusal_reason": "<string if mode=refusal>",

  "plan": {
    "table": "<table name>",
    "operation": "select" | "scan_then_write",
    "read": {
      "partiql": "<PartiQL SELECT>",
      "requires_scan": true | false,
      "index": "<GSI name>" | null,
      "projection": ["<PK name>", "<SK name>"] | ["*"]
    },
    "write": null | {
      "action": "update" | "delete",
      "per_item": {
        "partiql_template":
          "<Key-bounded PartiQL UPDATE/DELETE with placeholders {{PK}} and {{SK}} if SK exists>"
      }
    },
    "safety": {
      "needs_confirmation": true | false,
      "reason": "none" | "full_table_scan" | "multi_item_write"
    }
  }
}

EXAMPLES
CORRECT INSERT: INSERT INTO "Users" VALUE {'id': 1, 'name': 'bob'}
INCORRECT INSERT: INSERT INTO "Users" VALUE {'id': {{id}}, 'name': {{name}}}

CORRECT SINGLE-ITEM UPDATE (SQL Mode):
Request: "Update user 123 set status='active'" (assuming PK=id)
Result: {"mode": "sql", "statements": ["UPDATE \"Users\" SET \"status\"='active' WHERE \"id\"=123"]}

INCORRECT UPDATE (Wrong Placeholders): "partiql_template": "UPDATE \"Users\" SET \"status\"='active' WHERE \"id\"={{id}}"
CORRECT UPDATE (Plan Mode): "partiql_template": "UPDATE \"Users\" SET \"status\"='active' WHERE \"id\"={{PK}}"

REFUSAL EXAMPLES
Request: "What is the average age of users?" -> {"mode": "refusal", "refusal_reason": "Aggregations like AVG are not supported."}
Request: "Join Users and Orders tables" -> {"mode": "refusal", "refusal_reason": "Joins/Unions involving multiple tables are not supported."}
Request: "Update all users to have random passwords" -> {"mode": "refusal", "refusal_reason": "Dynamic value generation (random) is not supported for updates."}
Request: "Double the points for everyone" -> {"mode": "refusal", "refusal_reason": "Math operations on attributes (points = points * 2) are not supported."}
Request: "Set fullName to firstName + lastName" -> {"mode": "refusal", "refusal_reason": "String concatenation is not supported."}
Request: "Uppercase all usernames" -> {"mode": "refusal", "refusal_reason": "String transformation functions like UPPER() or LOWER() are not supported."}
Request: "Increase price by 10 for all items" -> {"mode": "refusal", "refusal_reason": "Mathematical updates (price = price + 10) are not supported in PartiQL."}
Request: "Set updated_at to current time" -> {"mode": "refusal", "refusal_reason": "Built-in timestamp functions like NOW() are not supported."}

STRICT DYNAMODB RULES
- UPDATE and DELETE must uniquely identify items using the FULL primary key.
  - PK-only table: WHERE must include PK equality.
  - PK+SK table: WHERE must include BOTH PK and SK equality.
- If the user asks to UPDATE or DELETE multiple items but does NOT provide keys,
  you MUST return operation="scan_then_write".
- If a filter does not use PK or a GSI partition key, set read.requires_scan=true. FULL TABLE SCANS ARE ALLOWED. Do not refuse.
- If read.requires_scan=true, set safety.needs_confirmation=true and safety.reason="full_table_scan".

LIMITATIONS (REFUSAL CRITERIA)
If the user requests any of the following, return mode="refusal" with a helpful refusal_reason:
1. Data Transformation: Operations like UPPER(), LOWER(), Concatenation, or math on attributes (e.g. "Double the price").
2. Aggregations: COUNT, AVG, SUM, MAX, MIN, GROUP BY.
3. DDL/Schema: CREATE, ALTER, DROP TABLE/INDEX.
4. Joins/Unions: Operations involving multiple tables.
5. Dynamic Value Generation: Requests asking to generate random values, UUIDs, or timestamps for EACH item during an update (e.g., "Set random password for everyone").
NOTE: Inefficient queries (Full Table Scans) are ALLOWED. Do not refuse them.

DECISION RULES
- If mode="refusal", set other fields to null/empty.
- For INSERT requests asking for random/dummy/sequential data, YOU (the AI) must generate the specific static values yourself and return them as a list of SQL statements in mode='sql'. Do NOT refuse. Do NOT use placeholders.
- CRITICAL: If the user asks for random/dynamic values in an UPDATE/DELETE (e.g. "set random password"), return mode="refusal". Do NOT attempt to use {{random}} placeholders.
- For INSERT operations (creating new items), ALWAYS use mode="sql" with fully specified statements. NEVER use a plan or templates for INSERT.
- When mode='sql', statements MUST NOT contain ANY placeholders like {{...}}. You must generate actual values (random or specific).
- If the request is a SELECT, UPDATE, or DELETE that uniquely identifies a SINGLE item using the FULL Primary Key (PK for simple tables; BOTH PK and SK for composite tables), return mode="sql".
- If a SELECT uses a partial key (e.g. only PK on a PK+SK table) or non-key attributes, it may return MULTIPLE items; return mode="plan" with operation="select".

PARTIQL RULES
- Double quotes for table/attribute names.
- Single quotes for string values.
- INSERT: INSERT INTO "Table" VALUE {...}
- UPDATE: UPDATE "Table" SET "attr" = 'val' OR UPDATE "Table" REMOVE "attr"
- Missing attribute: "attr" IS MISSING
- Check existence: attribute_exists("attr") NOT contains
- Remove field: UPDATE "Table" REMOVE "attr" WHERE ...
- partiql_template must ONLY contain {{PK}} and {{SK}} placeholders. Do NOT use placeholders like {{id}}, {{random}}, etc.
- Check list/string content: contains("tags", 'urgent'), begins_with("attr",'A')

Return ONLY the JSON object.
`, schemaDesc, question)

	body := NovaRequest{
		Messages: []NovaMessage{
			{
				Role: "user",
				Content: []NovaContent{
					{Text: prompt},
				},
			},
		},
		InferenceConfig: struct {
			MaxNewTokens int     `json:"max_new_tokens"`
			Temperature  float64 `json:"temperature"`
		}{
			MaxNewTokens: 5000, // Increased to 5000 (near max of 5,120 for Nova Lite)
			Temperature:  0,
		},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return LLMResult{}, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := a.Bedrock.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String("amazon.nova-lite-v1:0"),
		ContentType: aws.String("application/json"),
		Body:        payload,
	})
	if err != nil {
		return LLMResult{}, fmt.Errorf("invoke model: %w", err)
	}

	var response NovaResponse
	if err := json.Unmarshal(resp.Body, &response); err != nil {
		return LLMResult{}, fmt.Errorf("unmarshal response: %w", err)
	}

	if len(response.Output.Message.Content) == 0 {
		return LLMResult{}, fmt.Errorf("empty response from model")
	}

	rawText := strings.TrimSpace(response.Output.Message.Content[0].Text)

	// If the model ever wraps output in fences, strip them
	rawText = strings.TrimPrefix(rawText, "```json")
	rawText = strings.TrimPrefix(rawText, "```")
	rawText = strings.TrimSuffix(rawText, "```")
	rawText = strings.TrimSpace(rawText)

	log.Printf("Raw LLM Response: %s", rawText)

	var result LLMResult
	if err := json.Unmarshal([]byte(rawText), &result); err != nil {
		return LLMResult{}, fmt.Errorf("LLM JSON parse failed: %w; raw=%q", err, rawText)
	}

	log.Printf("Parsed Statements (Before Cleaning): %q", result.Statements)

	// Filter empty statements to avoid DynamoDB ValidationException
	if len(result.Statements) > 0 {
		var cleanStmts []string
		for _, s := range result.Statements {
			if strings.TrimSpace(s) != "" {
				cleanStmts = append(cleanStmts, s)
			}
		}
		result.Statements = cleanStmts
	}
	
	log.Printf("Final Statements: %q", result.Statements)

	// validations
	if result.Mode == "sql" && len(result.Statements) == 0 {
		return LLMResult{}, errors.New("mode=sql but no statements returned")
	}
	if result.Mode == "plan" && result.Plan == nil {
		return LLMResult{}, errors.New("mode=plan but plan is null")
	}
	if result.Plan != nil && result.Plan.Operation == "scan_then_write" && result.Plan.Write == nil {
		return LLMResult{}, errors.New("scan_then_write requires write block")
	}

	return result, nil
}

func FormatPartiQLValue(v any) (string, error) {
	switch t := v.(type) {
	case string:
		// PartiQL string literal
		return "'" + strings.ReplaceAll(t, "'", "''") + "'", nil
	case float64:
		// JSON numbers often unmarshal as float64
		// Format without trailing .0 when integer-like
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10), nil
		}
		return strconv.FormatFloat(t, 'f', -1, 64), nil
	case int:
		return strconv.Itoa(t), nil
	case int64:
		return strconv.FormatInt(t, 10), nil
	case bool:
		if t {
			return "true", nil
		}
		return "false", nil
	default:
		return "", fmt.Errorf("unsupported key type %T", v)
	}
}

func SubstituteKeys(tpl string, pkVal any, skVal any, hasSK bool) (string, error) {
	if !strings.Contains(tpl, "{{PK}}") {
		return "", fmt.Errorf("template missing {{PK}} placeholder")
	}

	pkStr, err := FormatPartiQLValue(pkVal)
	if err != nil {
		return "", fmt.Errorf("format PK: %w", err)
	}
	out := strings.ReplaceAll(tpl, "{{PK}}", pkStr)

	if hasSK {
		if !strings.Contains(tpl, "{{SK}}") {
			return "", fmt.Errorf("template missing {{SK}} placeholder for table with sort key")
		}
		skStr, err := FormatPartiQLValue(skVal)
		if err != nil {
			return "", fmt.Errorf("format SK: %w", err)
		}
		out = strings.ReplaceAll(out, "{{SK}}", skStr)
	}
	return out, nil
}
