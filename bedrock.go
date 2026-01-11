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
	"github.com/aws/aws-sdk-go-v2/config"
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
	Mode       string     `json:"mode"`
	Statements []string   `json:"statements"`
	Plan       *PlanBlock `json:"plan"`
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

func InvokeBedrock(ctx context.Context, question string, table Table) (LLMResult, error) {
	log.Printf("InvokeBedrock called with question: '%s' for table: '%s'", question, table.Name)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return LLMResult{}, fmt.Errorf("load aws config: %w", err)
	}

	client := bedrockruntime.NewFromConfig(cfg)

	// Construct schema description
	schemaDesc := fmt.Sprintf("Table Name: %s\nPartition Key: %s (Type: %s)\nSort Key: %s (Type: %s)\n", 
		table.Name, table.PK, table.PKType, table.SK, table.SKType)
	
	if len(table.GSIs) > 0 {
		schemaDesc += fmt.Sprintf("Global Secondary Indexes: %v\n", table.GSIs)
	}
	prompt := fmt.Sprintf(`
You are a DynamoDB expert. Your job is to produce a SAFE execution plan for DynamoDB.

Return EXACTLY ONE valid JSON object and nothing else (no markdown, no backticks, no explanations).

INPUTS
Schema (includes table name, PK/SK, GSIs):
%s

User request:
%s

STRICT DYNAMODB RULES
- UPDATE and DELETE must uniquely identify items using the FULL primary key.
  - PK-only table: WHERE must include PK equality.
  - PK+SK table: WHERE must include BOTH PK and SK equality.
- If the user asks to UPDATE or DELETE multiple items but does NOT provide keys,
  you MUST return operation="scan_then_write".
- If a filter does not use PK or a GSI partition key, set read.requires_scan=true.
- If read.requires_scan=true, set safety.needs_confirmation=true and safety.reason="full_table_scan".

OUTPUT JSON SCHEMA
{
  "mode": "sql" | "plan",
  "statements": ["<PartiQL>"],

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
      "action": "insert" | "update" | "delete",
      "per_item": {
        "partiql_template":
          "<Key-bounded PartiQL with placeholders {{PK}} and {{SK}} if SK exists>"
      }
    },
    "safety": {
      "needs_confirmation": true | false,
      "reason": "none" | "full_table_scan" | "multi_item_write"
    }
  }
}

DECISION RULES
- If the request can be satisfied with a SAFE single-step key-bounded PartiQL,
  return mode="sql" and populate "statements". Set "plan" to null.
- Otherwise return mode="plan", set "statements" to [],
  and produce a structured plan.
- For read-only requests that require scanning, use operation="select" and set "write": null.
- Only use operation="scan_then_write" when performing multi-item UPDATE or DELETE.

PARTIQL RULES
- Double quotes for table/attribute names.
- Single quotes for string values.
- INSERT: INSERT INTO "Table" VALUE {...}
- Missing attribute: "attr" IS MISSING
- contains("attr",'x'), begins_with("attr",'A')

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
			MaxNewTokens: 1000, // Increased for larger batches of statements
			Temperature:  0,
		},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return LLMResult{}, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := client.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
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

	var result LLMResult
	if err := json.Unmarshal([]byte(rawText), &result); err != nil {
		return LLMResult{}, fmt.Errorf("LLM JSON parse failed: %w; raw=%q", err, rawText)
	}

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
	pkStr, err := FormatPartiQLValue(pkVal)
	if err != nil {
		return "", fmt.Errorf("format PK: %w", err)
	}
	out := strings.ReplaceAll(tpl, "{{PK}}", pkStr)

	if hasSK {
		skStr, err := FormatPartiQLValue(skVal)
		if err != nil {
			return "", fmt.Errorf("format SK: %w", err)
		}
		out = strings.ReplaceAll(out, "{{SK}}", skStr)
	}
	return out, nil
}

