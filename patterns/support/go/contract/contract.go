package contract

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

type Evaluator func(map[string]any) (any, error)

type codedError interface {
	error
	Code() string
}

type fixture struct {
	Cases []struct {
		ID            string         `json:"id"`
		Input         map[string]any `json:"input"`
		Expected      any            `json:"expected"`
		ExpectedError *struct {
			Code string `json:"code"`
		} `json:"expected_error"`
	} `json:"cases"`
}

func Run(t *testing.T, contractPath string, evaluate Evaluator) {
	t.Helper()
	body, err := os.ReadFile(contractPath)
	if err != nil {
		t.Fatal(err)
	}
	var data fixture
	if err := json.Unmarshal(body, &data); err != nil {
		t.Fatal(err)
	}

	for _, testCase := range data.Cases {
		testCase := testCase
		t.Run(testCase.ID, func(t *testing.T) {
			result, err := evaluate(testCase.Input)
			if testCase.ExpectedError != nil {
				if err == nil {
					t.Fatalf("expected error %q, got result %#v", testCase.ExpectedError.Code, result)
				}
				coded, ok := err.(codedError)
				if !ok {
					t.Fatalf("error %T does not expose Code()", err)
				}
				if coded.Code() != testCase.ExpectedError.Code {
					t.Fatalf("error code = %q, want %q", coded.Code(), testCase.ExpectedError.Code)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			actualJSON, err := json.Marshal(result)
			if err != nil {
				t.Fatal(err)
			}
			expectedJSON, err := json.Marshal(testCase.Expected)
			if err != nil {
				t.Fatal(err)
			}
			if string(actualJSON) != string(expectedJSON) {
				t.Fatalf("result = %s, want %s", actualJSON, expectedJSON)
			}
		})
	}
}

func String(input map[string]any, key string) (string, error) {
	value, ok := input[key].(string)
	if !ok {
		return "", fmt.Errorf("%s must be a string", key)
	}
	return value, nil
}

func Number(input map[string]any, key string) (float64, error) {
	value, ok := input[key].(float64)
	if !ok {
		return 0, fmt.Errorf("%s must be a number", key)
	}
	return value, nil
}

