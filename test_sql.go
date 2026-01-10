package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func test() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Replace "us-east-1" and "test_table" with your actual region and table name for testing
	region := "us-east-1"
	tableName := "test_table"

	op := Operation{
		expression: fmt.Sprintf("SELECT * FROM %s", tableName),
		params:     []types.AttributeValue{},
	}

	fmt.Printf("Executing query on table: %s in region: %s...\n", tableName, region)
	items, err := SqlQuery(ctx, region, op)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}

	fmt.Printf("Query successful! Items found: %d\n", len(items))
	for i, item := range items {
		fmt.Printf("Item %d: %v\n", i, item)
	}
}

