package yamlutil_test

import (
	"fmt"

	"github.com/dshills/gocreator/internal/yamlutil"
)

// ExampleUnmarshal_indentationError demonstrates enhanced error output for indentation errors
func ExampleUnmarshal_indentationError() {
	yaml := `
name: test
  version: 1.0
config:
  timeout: 30s
`

	var result map[string]interface{}
	err := yamlutil.Unmarshal([]byte(yaml), &result)

	if err != nil {
		fmt.Println("Error occurred:")
		fmt.Println(err.Error())
	}

	// Output shows:
	// - Line number (3)
	// - Context showing surrounding lines with arrow pointing to error
	// - Helpful suggestion about indentation
}

// ExampleUnmarshal_typeMismatch demonstrates enhanced error output for type mismatches
func ExampleUnmarshal_typeMismatch() {
	yaml := `
name: "test"
config:
  max_parallel: 4
  timeout: "not a duration"
  retries: "three"
`

	type Config struct {
		Name   string `yaml:"name"`
		Config struct {
			MaxParallel int    `yaml:"max_parallel"`
			Timeout     string `yaml:"timeout"`
			Retries     int    `yaml:"retries"`
		} `yaml:"config"`
	}

	var result Config
	err := yamlutil.Unmarshal([]byte(yaml), &result)

	if err != nil {
		fmt.Println("Error occurred:")
		fmt.Println(err.Error())
	}

	// Output shows:
	// - Exact line where type mismatch occurs (line 6)
	// - Context showing the problematic line
	// - Suggestion about checking data types
}

// ExampleUnmarshal_tabIndentation demonstrates detection of tab characters
func ExampleUnmarshal_tabIndentation() {
	yaml := "name: test\n\tversion: 1.0\n"

	var result map[string]interface{}
	err := yamlutil.Unmarshal([]byte(yaml), &result)

	if err != nil {
		fmt.Println("Error occurred:")
		fmt.Println(err.Error())
	}

	// Output shows:
	// - Line number with the tab character
	// - Clear suggestion to replace tabs with spaces
}

// ExampleUnmarshalStrict_unknownField demonstrates strict mode rejecting unknown fields
func ExampleUnmarshalStrict_unknownField() {
	yaml := `
schema_version: "1.0"
name: "test"
unknown_field: "value"
extra_field: 123
`

	type Config struct {
		SchemaVersion string `yaml:"schema_version"`
		Name          string `yaml:"name"`
	}

	var result Config
	err := yamlutil.UnmarshalStrict([]byte(yaml), &result)

	if err != nil {
		fmt.Println("Error occurred:")
		fmt.Println(err.Error())
	}

	// Output shows:
	// - Line number of the unknown field
	// - Suggestion to check field names in documentation
}

// ExampleUnmarshal_missingColon demonstrates missing colon detection
func ExampleUnmarshal_missingColon() {
	yaml := `
name test
version: "1.0"
`

	var result map[string]interface{}
	err := yamlutil.Unmarshal([]byte(yaml), &result)

	if err != nil {
		fmt.Println("Error occurred:")
		fmt.Println(err.Error())
	}

	// Output shows:
	// - Line number where colon is expected
	// - Context showing the line
	// - Suggestion about YAML syntax requirements
}

// ExampleValidate demonstrates YAML validation without unmarshaling to specific type
func ExampleValidate() {
	yaml := `
name: test
config:
  timeout: 30s
  retries: 3
  max_parallel: 4
`

	err := yamlutil.Validate([]byte(yaml))

	if err != nil {
		fmt.Println("Validation failed:")
		fmt.Println(err.Error())
	} else {
		fmt.Println("YAML is valid")
	}

	// Output:
	// YAML is valid
}
