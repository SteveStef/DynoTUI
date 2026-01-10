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

// InvokeBedrock calls AWS Bedrock (Amazon Nova Lite) to generate a DynamoDB PartiQL query.
func InvokeBedrock(ctx context.Context, question string, table Table) (string, error) {
	log.Printf("InvokeBedrock called with question: '%s' for table: '%s'", question, table.Name)

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"))
	if err != nil {
		return "", fmt.Errorf("load aws config: %w", err)
	}

	client := bedrockruntime.NewFromConfig(cfg)

	// Construct schema description
	schemaDesc := fmt.Sprintf("Table Name: %s\nPartition Key: %s\nSort Key: %s\n", table.Name, table.PK, table.SK)
	if len(table.GSIs) > 0 {
		schemaDesc += fmt.Sprintf("Global Secondary Indexes: %v\n", table.GSIs)
	}

	prompt := fmt.Sprintf(`You are a DynamoDB expert. Generate a PartiQL query for the following request.
Schema:
%s

Request: %s

Rules:
1. Return ONLY the raw SQL query. No markdown, no explanations.
2. Use double quotes for table names and column names if reserved words.
3. Use single quotes for string values.
4. Ensure the query is valid PartiQL for DynamoDB.
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
			MaxNewTokens: 200,
			Temperature:  0,
		},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	resp, err := client.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String("amazon.nova-lite-v1:0"),
		ContentType: aws.String("application/json"),
		Body:        payload,
	})
	if err != nil {
		return "", fmt.Errorf("invoke model: %w", err)
	}

	var response NovaResponse
	if err := json.Unmarshal(resp.Body, &response); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if len(response.Output.Message.Content) == 0 {
		return "", fmt.Errorf("empty response from model")
	}

	sql := response.Output.Message.Content[0].Text

	// Clean up the response (remove markdown code blocks if present)
	sql = strings.TrimSpace(sql)
	sql = strings.ReplaceAll(sql, "```sql", "")
	sql = strings.ReplaceAll(sql, "```", "")
	sql = strings.TrimSpace(sql)

	return sql, nil
}
