package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

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

// InvokeBedrock calls AWS Bedrock (Amazon Nova Lite) to generate DynamoDB PartiQL queries.
func InvokeBedrock(ctx context.Context, question string, table Table) ([]string, error) {
	log.Printf("InvokeBedrock called with question: '%s' for table: '%s'", question, table.Name)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := bedrockruntime.NewFromConfig(cfg)

	// Construct schema description
	schemaDesc := fmt.Sprintf("Table Name: %s\nPartition Key: %s\nSort Key: %s\n", table.Name, table.PK, table.SK)
	if len(table.GSIs) > 0 {
		schemaDesc += fmt.Sprintf("Global Secondary Indexes: %v\n", table.GSIs)
	}

	prompt := fmt.Sprintf(`You are a DynamoDB expert. Generate PartiQL queries for the following request.
Schema:
%s

Request: %s

Examples:
- "Show me users with age 25": SELECT * FROM "Users" WHERE "age" = 25
- "Add a user with id 1 and name bob": INSERT INTO "Users" VALUE {'id': 1, 'name': 'bob'}
- "Update user 1 name to alice": UPDATE "Users" SET "name" = 'alice' WHERE "id" = 1
- "Delete user 1": DELETE FROM "Users" WHERE "id" = 1

Rules:
1. Return ONLY the raw SQL queries, separated by semicolons if multiple.
2. No markdown, no explanations.
3. Use double quotes for table names and column names if reserved words or case-sensitive.
4. Use single quotes for string values.
5. For INSERT, use "VALUE {'key': 'val'}" syntax (singular VALUE, no parentheses).
6. Ensure the queries are valid PartiQL for DynamoDB.
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
			MaxNewTokens: 300, // Increased for multiple statements
			Temperature:  0,
		},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := client.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String("amazon.nova-lite-v1:0"),
		ContentType: aws.String("application/json"),
		Body:        payload,
	})
	if err != nil {
		return nil, fmt.Errorf("invoke model: %w", err)
	}

	var response NovaResponse
	if err := json.Unmarshal(resp.Body, &response); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if len(response.Output.Message.Content) == 0 {
		return nil, fmt.Errorf("empty response from model")
	}

	rawText := response.Output.Message.Content[0].Text

	// Clean up the response
	rawText = strings.TrimSpace(rawText)
	rawText = strings.ReplaceAll(rawText, "```sql", "")
	rawText = strings.ReplaceAll(rawText, "```", "")
	rawText = strings.TrimSpace(rawText)

	// Split by semicolon and filter empty strings
	var statements []string
	parts := strings.Split(rawText, ";")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			statements = append(statements, p)
		}
	}

	return statements, nil
}
