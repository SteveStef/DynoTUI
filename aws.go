package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// ListAllTables returns all DynamoDB table names in the configured account/region.
func ListAllTables(ctx context.Context, region string) ([]string, error) {
	// Loads credentials from the standard AWS chain:
	// env vars, shared config (~/.aws), ECS/EC2 role, etc.
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
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
func ListTablesWithDetails(ctx context.Context, region string) ([]TableDetails, error) {
	names, err := ListAllTables(ctx, region)
	if err != nil {
		return nil, err
	}

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, err
	}
	client := dynamodb.NewFromConfig(cfg)

	var tables []TableDetails
	for _, name := range names {
		resp, err := client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
			TableName: aws.String(name),
		})
		if err != nil {
			// Skip tables we can't describe or handle error? 
			// For now, let's just log print and continue or return error. 
			// Best to return error for TUI feedback.
			return nil, fmt.Errorf("describe table %s: %w", name, err)
		}
		
		t := resp.Table
		details := TableDetails{
			Name:      *t.TableName,
			Region:    region,
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

	return tables, nil
}

// ScanTable fetches all items from the specified DynamoDB table.
func ScanTable(ctx context.Context, region string, tableName string) ([]map[string]interface{}, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := dynamodb.NewFromConfig(cfg)
	var items []map[string]interface{}

	paginator := dynamodb.NewScanPaginator(client, &dynamodb.ScanInput{
		TableName: aws.String(tableName),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("scan page: %w", err)
		}
		
		for _, item := range page.Items {
			var unmarshalledItem map[string]interface{}
			if err := attributevalue.UnmarshalMap(item, &unmarshalledItem); err != nil {
				// Log error or skip? For now, we'll try to continue
				continue 
			}
			items = append(items, unmarshalledItem)
		}

		// Safety break for development: limit to 1000 items
		if len(items) >= 1000 {
			break
		}
	}

	return items, nil
}

// PutItem uploads an item to DynamoDB (Update/Insert)
func PutItem(ctx context.Context, region string, tableName string, item map[string]interface{}) error {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
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
func DeleteItem(ctx context.Context, region string, tableName string, key map[string]interface{}) error {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
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


