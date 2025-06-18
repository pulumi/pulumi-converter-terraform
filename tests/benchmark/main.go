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

func resultSummary(results map[string]*benchmarkResult) string {
	buf := bytes.Buffer{}
	total := len(results)
	type summary struct {
		convertSuccesses int
		planSuccesses    int
		applySuccesses   int
		assertSuccesses  int
	}
	res := summary{}
	for _, v := range results {
		if v.convertSuccess {
			res.convertSuccesses++
		}
		if v.planSuccess {
			res.planSuccesses++
		}
		if v.applySuccess {
			res.applySuccesses++
		}
		if v.assertSuccess {
			res.assertSuccesses++
		}
	}
	buf.WriteString(fmt.Sprintf("total: %d\n", total))
	buf.WriteString(fmt.Sprintf("convertSuccesses: %d (%d%%)\n", res.convertSuccesses, res.convertSuccesses*100/total))
	buf.WriteString(fmt.Sprintf("planSuccesses: %d (%d%%)\n", res.planSuccesses, res.planSuccesses*100/total))
	buf.WriteString(fmt.Sprintf("applySuccesses: %d (%d%%)\n", res.applySuccesses, res.applySuccesses*100/total))
	buf.WriteString(fmt.Sprintf("assertSuccesses: %d (%d%%)\n", res.assertSuccesses, res.assertSuccesses*100/total))
	return buf.String()
}

func main() {
	tfResults := runTofuBenchmarks(testCases)
	fmt.Printf("tfResults:\n%s", formatResults(tfResults))
	claudeResults := runPulumiBenchmarks(testCases, runClaudeConvert)
	fmt.Printf("claudeResults:\n%s", formatResults(claudeResults))
	pulumiResultsTs := runPulumiBenchmarks(testCases, runPulumiConvertTS)
	fmt.Printf("pulumiResultsTs:\n%s", formatResults(pulumiResultsTs))
	pulumiResultsPy := runPulumiBenchmarks(testCases, runPulumiConvertPy)
	fmt.Printf("pulumiResultsPy:\n%s", formatResults(pulumiResultsPy))
	pulumiResultsGo := runPulumiBenchmarks(testCases, runPulumiConvertGo)
	fmt.Printf("pulumiResultsGo:\n%s", formatResults(pulumiResultsGo))
	pulumiResultsCs := runPulumiBenchmarks(testCases, runPulumiConvertCs)
	fmt.Printf("pulumiResultsCs:\n%s", formatResults(pulumiResultsCs))
	pulumiResultsJava := runPulumiBenchmarks(testCases, runPulumiConvertJava)
	fmt.Printf("pulumiResultsJava:\n%s", formatResults(pulumiResultsJava))
	pulumiResultsYaml := runPulumiBenchmarks(testCases, runPulumiConvertYaml)
	fmt.Printf("pulumiResultsYaml:\n%s", formatResults(pulumiResultsYaml))
	fmt.Println("--------------------------------")
	fmt.Printf("tfResults:\n%s", resultSummary(tfResults))
	fmt.Printf("claudeResults:\n%s", resultSummary(claudeResults))
	fmt.Printf("pulumiResultsTs:\n%s", resultSummary(pulumiResultsTs))
	fmt.Printf("pulumiResultsPy:\n%s", resultSummary(pulumiResultsPy))
	fmt.Printf("pulumiResultsGo:\n%s", resultSummary(pulumiResultsGo))
	fmt.Printf("pulumiResultsCs:\n%s", resultSummary(pulumiResultsCs))
	fmt.Printf("pulumiResultsJava:\n%s", resultSummary(pulumiResultsJava))
	fmt.Printf("pulumiResultsYaml:\n%s", resultSummary(pulumiResultsYaml))
}
