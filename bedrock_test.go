package main

import (
	"context"
	"fmt"
	"testing"
)

type TestCase struct {
	question     string
	expectedMode string
}

// TestBedrockPrompts runs a suite of LLM prompt tests.
// Run with: go test -v -run TestBedrockPrompts
func TestBedrockPrompts(t *testing.T) {
	cases := []TestCase{
		// {"Get the user with id 2 and age 29", "sql"},
		// {"Add a new user with id 55, age 30 and name john", "sql"},
		// {"Find all items where name is banana", "plan"}, // Likely scan -> plan

		// Supported - Direct SQL (Single Item)
		//{"Update user with id 123 and age 30, set status to 'active'", "sql"},
		//{"Delete the item with id 123 and age 30", "sql"},

		// // Supported - Bulk Updates/Deletes (Plan)
		// {"Set status to 'archived' for all items where age > 50", "plan"},
		//{"Delete all items where status is 'inactive'", "plan"},
		//
		// // Supported - Batch Inserts (Explicit SQL)
		//{"Add 3 items: {id: 1, age: 20, name: 'a'}, {id: 2, age: 21, name: 'b'}, {id: 3, age: 22, name: 'c'}", "sql"},
		//
		// // Unsupported - Refusals
		//{"Set a random password for everyone", "refusal"},
		//{"What is the average age?", "refusal"},
		// {"Uppercase all usernames", "refusal"},
		// {"Join users with orders", "refusal"},
		// {"Create a new table", "refusal"},
	}

	ctx := context.TODO()
	// Note: NewAWS requires valid AWS configuration in environment
	api, err := NewAWS(ctx)
	if err != nil {
		t.Fatalf("Failed to initialize AWS client: %v", err)
	}

	// Mock Table Definition
	tbl := Table{
		Name:      "test_table2",
		PK:        "id",
		PKType:    "N",
		SK:        "age",
		SKType:    "N",
		Region:    "us-east-1",
		ItemCount: 100,
		GSIs:      []string{},
		Status:    "ACTIVE",
	}

	fmt.Println("Starting LLM Prompt Tests...")
	fmt.Println("---------------------------------------------------")

	for _, cas := range cases {
		fmt.Printf("QUERY: %s\n", cas.question)

		result, err := api.InvokeBedrock(ctx, cas.question, tbl)
		if err != nil {
			t.Errorf("InvokeBedrock failed for '%s': %v", cas.question, err)
			continue
		}

		fmt.Printf(" -> Mode: %s\n", result.Mode)
		if result.Reason != "" {
			fmt.Printf(" -> Reason: %s\n", result.Reason)
		}

		// Validation logic
		if result.Mode != cas.expectedMode {
			t.Errorf("Expected mode '%s', got '%s'", cas.expectedMode, result.Mode)
		}
		if result.Mode == "refusal" {
			fmt.Printf(" -> Reason: %s\n", result.RefusalReason)
		} else if result.Mode == "sql" {
			for _, s := range result.Statements {
				fmt.Printf(" -> SQL: %s\n", s)
			}
		} else if result.Mode == "plan" {
			fmt.Printf(" -> Plan Op: %s\n", result.Plan.Operation)
			fmt.Printf(" -> Read: %s\n", result.Plan.Read.Partiql)
			if result.Plan.Write != nil {
				fmt.Printf(" -> Write: %s\n", result.Plan.Write.PerItem.PartiqlTemplate)
			}
		}

		fmt.Println("---------------------------------------------------")
	}
}
