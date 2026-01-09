package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
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
			items = append(items, unmarshalItem(item))
		}

		// Safety break for development: limit to 1000 items
		if len(items) >= 1000 {
			break
		}
	}

	return items, nil
}

// unmarshalAttributeValue converts DynamoDB AttributeValue to native Go types
func unmarshalAttributeValue(av types.AttributeValue) interface{} {
	switch v := av.(type) {
	case *types.AttributeValueMemberS:
		return v.Value
	case *types.AttributeValueMemberN:
		return v.Value // Returns string representation of number
	case *types.AttributeValueMemberBOOL:
		return v.Value
	case *types.AttributeValueMemberM:
		// Recursive for Maps
		out := make(map[string]interface{})
		for k, val := range v.Value {
			out[k] = unmarshalAttributeValue(val)
		}
		return out
	case *types.AttributeValueMemberL:
		// Recursive for Lists
		var out []interface{}
		for _, val := range v.Value {
			out = append(out, unmarshalAttributeValue(val))
		}
		return out
	case *types.AttributeValueMemberNULL:
		return nil
	default:
		return nil
	}
}

// Helper to convert the whole item map
func unmarshalItem(item map[string]types.AttributeValue) map[string]interface{} {
	out := make(map[string]interface{})
	for k, v := range item {
		out[k] = unmarshalAttributeValue(v)
	}
	return out
}

func main() {
	ctx := context.Background()
	tables, err := ListAllTables(ctx, "us-east-1")
	items, error := ScanTable(ctx, "us-east-1", tables[0])
	if err != nil { panic(err) }
	if error != nil { panic(error) }
	fmt.Println(tables)
	fmt.Println(items[0])
}

