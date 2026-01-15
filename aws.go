package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type AWS struct {
	Dynamo   *dynamodb.Client
	Bedrock  *bedrockruntime.Client
	Region   string
	AccountID string
}

func NewAWS(ctx context.Context) (*AWS, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	stsClient := sts.NewFromConfig(cfg)
	identity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	var accountID string
	if err == nil && identity.Account != nil {
		accountID = *identity.Account
	} else {
		accountID = "unknown"
	}

	return &AWS{
		Dynamo:    dynamodb.NewFromConfig(cfg),
		Bedrock:   bedrockruntime.NewFromConfig(cfg),
		Region:    cfg.Region,
		AccountID: accountID,
	}, nil
}

func (a *AWS) SqlQuery(ctx context.Context, operation Operation) ([]map[string]interface{}, error) {
	input := &dynamodb.ExecuteStatementInput{
		Statement: aws.String(operation.expression),
	}

	if len(operation.params) > 0 {
		input.Parameters = operation.params
	}

	result, err := a.Dynamo.ExecuteStatement(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("PartiQL execution failed: %w", err)
	}

	var items []map[string]interface{}
	if err := attributevalue.UnmarshalListOfMaps(result.Items, &items); err != nil {
		return nil, fmt.Errorf("unmarshal items: %w", err)
	}

	return items, nil
}

func (a *AWS) BatchSqlQuery(ctx context.Context, statements []string) ([]map[string]interface{}, error) {
	var allItems []map[string]interface{}
	var errorMsgs []string

	// Chunk size for BatchExecuteStatement is 25
	chunkSize := 25

	for i := 0; i < len(statements); i += chunkSize {
		end := i + chunkSize
		if end > len(statements) {
			end = len(statements)
		}

		chunk := statements[i:end]
		var batchInputs []types.BatchStatementRequest
		for _, sql := range chunk {
			batchInputs = append(batchInputs, types.BatchStatementRequest{
				Statement: aws.String(sql),
			})
		}

		result, err := a.Dynamo.BatchExecuteStatement(ctx, &dynamodb.BatchExecuteStatementInput{
			Statements: batchInputs,
		})

		if err != nil {
			return allItems, fmt.Errorf("batch execution failed at chunk %d-%d: %w", i, end, err)
		}

		// Process responses for this chunk
		for j, resp := range result.Responses {
			if resp.Error != nil {
				// Global index of the statement
				stmtIdx := i + j + 1
				msg := fmt.Sprintf("Statement %d failed: %s - %s", stmtIdx, resp.Error.Code, *resp.Error.Message)
				log.Println(msg)
				errorMsgs = append(errorMsgs, msg)
				continue
			}
			
			if resp.Item != nil {
				var item map[string]interface{}
				if err := attributevalue.UnmarshalMap(resp.Item, &item); err == nil {
					allItems = append(allItems, item)
				}
			}
		}
	}

	if len(errorMsgs) > 0 {
		return allItems, fmt.Errorf("Batch execution encountered %d errors:\n%s", len(errorMsgs), strings.Join(errorMsgs, "\n"))
	}

	return allItems, nil
}

// ListAllTables returns all DynamoDB table names in the configured account/region.
func (a *AWS) ListAllTables(ctx context.Context) ([]string, error) {
	var out []string
	var start *string

	for {
		resp, err := a.Dynamo.ListTables(ctx, &dynamodb.ListTablesInput{
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
	PKType    string
	SK        string
	SKType    string
	Region    string
	ItemCount int64
	GSIs      []string
	Status    string
}

// ListTablesWithDetails fetches names and then calls DescribeTable for each to get schema info.
func (a *AWS) ListTablesWithDetails(ctx context.Context) ([]TableDetails, string, string, error) {
	names, err := a.ListAllTables(ctx)
	if err != nil {
		return nil, "", "", err
	}

	var tables []TableDetails
	for _, name := range names {
		resp, err := a.Dynamo.DescribeTable(ctx, &dynamodb.DescribeTableInput{
			TableName: aws.String(name),
		})
		if err != nil {
			return nil, a.Region, a.AccountID, fmt.Errorf("describe table %s: %w", name, err)
		}
		
		t := resp.Table
		details := TableDetails{
			Name:      *t.TableName,
			Region:    a.Region,
			ItemCount: 0,
			Status:    string(t.TableStatus),
		}
		if t.ItemCount != nil {
			details.ItemCount = *t.ItemCount
		}

		// Build map of attribute types
		attrTypes := make(map[string]string)
		for _, ad := range t.AttributeDefinitions {
			attrTypes[*ad.AttributeName] = string(ad.AttributeType)
		}

		// Parse Key Schema
		for _, k := range t.KeySchema {
			if k.KeyType == types.KeyTypeHash {
				details.PK = *k.AttributeName
				if t, ok := attrTypes[details.PK]; ok {
					details.PKType = t
				}
			} else if k.KeyType == types.KeyTypeRange {
				details.SK = *k.AttributeName
				if t, ok := attrTypes[details.SK]; ok {
					details.SKType = t
				}
			}
		}

		// Parse GSIs
		for _, gsi := range t.GlobalSecondaryIndexes {
			details.GSIs = append(details.GSIs, *gsi.IndexName)
		}

		// Get Real-Time Count (Scan with Count)
		scanOut, err := a.Dynamo.Scan(ctx, &dynamodb.ScanInput{
			TableName: aws.String(name),
			Select:    types.SelectCount,
		})
		if err == nil {
			details.ItemCount = int64(scanOut.Count)
		}

		tables = append(tables, details)
	}

	return tables, a.Region, a.AccountID, nil
}

// ScanTable fetches items from DynamoDB. It accepts an exclusiveStartKey for pagination.
// It returns up to 1000 items and the LastEvaluatedKey for the next page.
func (a *AWS) ScanTable(ctx context.Context, tableName string, startKey map[string]types.AttributeValue) ([]map[string]interface{}, map[string]types.AttributeValue, error) {
	var items []map[string]interface{}
	var lastKey map[string]types.AttributeValue = startKey

	// Loop until we have 1000 items or no more pages
	for {
		input := &dynamodb.ScanInput{
			TableName:         aws.String(tableName),
			ExclusiveStartKey: lastKey,
			Limit:             aws.Int32(1000 - int32(len(items))), // Request only what we need to reach 1000
		}

		resp, err := a.Dynamo.Scan(ctx, input)
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
func (a *AWS) PutItem(ctx context.Context, tableName string, item map[string]interface{}) error {
	// Marshal Go map to DynamoDB AttributeValue map
	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return fmt.Errorf("marshal item: %w", err)
	}

	_, err = a.Dynamo.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      av,
	})
	if err != nil {
		return fmt.Errorf("put item: %w", err)
	}

	return nil
}

// DeleteItem deletes an item from DynamoDB
func (a *AWS) DeleteItem(ctx context.Context, tableName string, key map[string]interface{}) error {
	// Marshal Go map to DynamoDB AttributeValue map for key
	av, err := attributevalue.MarshalMap(key)
	if err != nil {
		return fmt.Errorf("marshal key: %w", err)
	}

	_, err = a.Dynamo.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(tableName),
		Key:       av,
	})
	if err != nil {
		return fmt.Errorf("delete item: %w", err)
	}

	return nil
}
