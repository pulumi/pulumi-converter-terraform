package main

import (
	"bytes"
	"fmt"
)

var testCases = []testCase{
	{
		name: "random_simple",
		dir:  "programs/random_simple",
		assertion: func(output map[string]any) error {
			if output["name"] == nil {
				return fmt.Errorf("name is nil")
			}
			if len(output["name"].(string)) == 0 {
				return fmt.Errorf("name is empty")
			}
			return nil
		},
	},
	{
		name: "aws_bucket",
		dir:  "programs/aws_bucket",
		assertion: func(output map[string]any) error {
			if output["url"] == nil {
				return fmt.Errorf("url is nil")
			}
			if len(output["url"].(string)) == 0 {
				return fmt.Errorf("url is empty")
			}
			out, err := run(".", "aws", "s3", "cp", output["url"].(string), "-")
			if err != nil {
				return fmt.Errorf("failed to copy object: %w", err)
			}
			if string(out) != "hi" {
				return fmt.Errorf("expected 'hi', got %s", string(out))
			}

			return nil
		},
	},
}

func formatResults(results map[string]*benchmarkResult) string {
	buf := bytes.Buffer{}
	for k, v := range results {
		buf.WriteString(fmt.Sprintf("%s: %+v\n", k, v))
	}
	return buf.String()
}

func main() {
	tfResults := runTofuBenchmarks(testCases)
	fmt.Printf("tfResults: %s", formatResults(tfResults))
	pulumiResults := runPulumiBenchmarks(testCases, runPulumiConvert)
	fmt.Printf("pulumiResults: %s", formatResults(pulumiResults))
	claudeResults := runPulumiBenchmarks(testCases, runClaudeConvert)
	fmt.Printf("claudeResults: %s", formatResults(claudeResults))
}
