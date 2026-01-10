package main

import (
	"context"
	"fmt"
	"log"
	// "time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

func SqlQuery(ctx context.Context, operation Operation) ([]map[string]interface{}, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := dynamodb.NewFromConfig(cfg)

	input := &dynamodb.ExecuteStatementInput{
		Statement: aws.String(operation.expression),
	}

	if len(operation.params) > 0 {
		input.Parameters = operation.params
	}

	result, err := client.ExecuteStatement(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("PartiQL execution failed: %w", err)
	}

	var items []map[string]interface{}
	if err := attributevalue.UnmarshalListOfMaps(result.Items, &items); err != nil {
		return nil, fmt.Errorf("unmarshal items: %w", err)
	}

	return items, nil
}

func BatchSqlQuery(ctx context.Context, statements []string) ([]map[string]interface{}, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := dynamodb.NewFromConfig(cfg)
	
	var batchInputs []types.BatchStatementRequest
	for _, sql := range statements {
		batchInputs = append(batchInputs, types.BatchStatementRequest{
			Statement: aws.String(sql),
		})
	}

	// DynamoDB BatchExecuteStatement allows up to 25 statements
	if len(batchInputs) > 25 {
		return nil, fmt.Errorf("too many statements for batch execution (max 25)")
	}

	result, err := client.BatchExecuteStatement(ctx, &dynamodb.BatchExecuteStatementInput{
		Statements: batchInputs,
	})

	if err != nil {
		return nil, fmt.Errorf("batch execution failed: %w", err)
	}

	// Aggregate all responses
	var allItems []map[string]interface{}
	for i, resp := range result.Responses {
		if resp.Error != nil {
			log.Printf("Batch statement %d error: %s - %s", i, resp.Error.Code, *resp.Error.Message)
			continue
		}
		
		if resp.Item != nil {
			var item map[string]interface{}
			if err := attributevalue.UnmarshalMap(resp.Item, &item); err == nil {
				allItems = append(allItems, item)
			}
		}
	}

	return allItems, nil
}

// ListAllTables returns all DynamoDB table names in the configured account/region.
func ListAllTables(ctx context.Context) ([]string, error) {
	// Loads credentials from the standard AWS chain:
	// env vars, shared config (~/.aws), ECS/EC2 role, etc.
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := dynamodb.NewFromConfig(cfg)

	var out []string
	var start *string

	for {
		resp, err := client.ListTables(ctx, &dynamodb.ListTablesInput{
			ExclusiveStartTableName: start,
			Limit:                  aws.Int32(100), // max is 100
		})
		if err != nil {
			return nil, fmt.Errorf("list tables: %w", err)
		}

		out = append(out, resp.TableNames...)

		if resp.LastEvaluatedTableName == nil || *resp.LastEvaluatedTableName == "" {
			break
		}
		start = resp.LastEvaluatedTableName
	}

	return out, nil
}

type TableDetails struct {
	Name      string
	PK        string
	SK        string
	Region    string
	ItemCount int64
	GSIs      []string
	Status    string
}

// ListTablesWithDetails fetches names and then calls DescribeTable for each to get schema info.
func ListTablesWithDetails(ctx context.Context) ([]TableDetails, string, string, error) {
	names, err := ListAllTables(ctx)
	if err != nil {
		return nil, "", "", err
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, "", "", err
	}
	
	// Get Account ID via STS
	stsClient := sts.NewFromConfig(cfg)
	identity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	var accountID string
	if err == nil && identity.Account != nil {
		accountID = *identity.Account
	} else {
		accountID = "unknown"
	}

	client := dynamodb.NewFromConfig(cfg)
	
	resolvedRegion := cfg.Region

	var tables []TableDetails
	for _, name := range names {
		resp, err := client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
			TableName: aws.String(name),
		})
		if err != nil {
			// Skip tables we can't describe or handle error? 
			// For now, let's just log print and continue or return error. 
			// Best to return error for TUI feedback.
			return nil, resolvedRegion, accountID, fmt.Errorf("describe table %s: %w", name, err)
		}
		
		t := resp.Table
		details := TableDetails{
			Name:      *t.TableName,
			Region:    resolvedRegion,
			ItemCount: 0,
			Status:    string(t.TableStatus),
		}
		if t.ItemCount != nil {
			details.ItemCount = *t.ItemCount
		}

		// Parse Key Schema
		for _, k := range t.KeySchema {
			if k.KeyType == types.KeyTypeHash {
				details.PK = *k.AttributeName
			} else if k.KeyType == types.KeyTypeRange {
				details.SK = *k.AttributeName
			}
		}

		// Parse GSIs
		for _, gsi := range t.GlobalSecondaryIndexes {
			details.GSIs = append(details.GSIs, *gsi.IndexName)
		}

		// Get Real-Time Count (Scan with Count)
		// valid for small dev tables; caution on large prod tables
		scanOut, err := client.Scan(ctx, &dynamodb.ScanInput{
			TableName: aws.String(name),
			Select:    types.SelectCount,
		})
		if err == nil {
			details.ItemCount = int64(scanOut.Count)
		}

		tables = append(tables, details)
	}

	return tables, resolvedRegion, accountID, nil
}

// ScanTable fetches items from DynamoDB. It accepts an exclusiveStartKey for pagination.
// It returns up to 1000 items and the LastEvaluatedKey for the next page.
func ScanTable(ctx context.Context, tableName string, startKey map[string]types.AttributeValue) ([]map[string]interface{}, map[string]types.AttributeValue, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("load aws config: %w", err)
	}

	client := dynamodb.NewFromConfig(cfg)
	var items []map[string]interface{}
	var lastKey map[string]types.AttributeValue = startKey

	// Loop until we have 1000 items or no more pages
	for {
		input := &dynamodb.ScanInput{
			TableName:         aws.String(tableName),
			ExclusiveStartKey: lastKey,
			Limit:             aws.Int32(1000 - int32(len(items))), // Request only what we need to reach 1000
		}

		resp, err := client.Scan(ctx, input)
		if err != nil {
			return nil, nil, fmt.Errorf("scan failed: %w", err)
		}

		for _, item := range resp.Items {
			var unmarshalledItem map[string]interface{}
			if err := attributevalue.UnmarshalMap(item, &unmarshalledItem); err == nil {
				items = append(items, unmarshalledItem)
			}
		}

		lastKey = resp.LastEvaluatedKey

		// Stop if we have enough items or if there are no more items to scan
		if len(items) >= 1000 || lastKey == nil {
			break
		}
	}

	return items, lastKey, nil
}

// PutItem uploads an item to DynamoDB (Update/Insert)
func PutItem(ctx context.Context, tableName string, item map[string]interface{}) error {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("load aws config: %w", err)
	}

	client := dynamodb.NewFromConfig(cfg)

	// Marshal Go map to DynamoDB AttributeValue map
	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return fmt.Errorf("marshal item: %w", err)
	}

	_, err = client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      av,
	})
	if err != nil {
		return fmt.Errorf("put item: %w", err)
	}

	return nil
}

// DeleteItem deletes an item from DynamoDB
func DeleteItem(ctx context.Context, tableName string, key map[string]interface{}) error {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("load aws config: %w", err)
	}

	client := dynamodb.NewFromConfig(cfg)

	// Marshal Go map to DynamoDB AttributeValue map for key
	av, err := attributevalue.MarshalMap(key)
	if err != nil {
		return fmt.Errorf("marshal key: %w", err)
	}

	_, err = client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(tableName),
		Key:       av,
	})
	if err != nil {
		return fmt.Errorf("delete item: %w", err)
	}

	return nil
}


