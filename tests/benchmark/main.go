package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

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
		assertTotal      int
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
		for _, assertSuccess := range v.assertSuccesses {
			if assertSuccess {
				res.assertSuccesses++
			}
		}
		res.assertTotal += len(v.assertSuccesses)
	}
	buf.WriteString(fmt.Sprintf("total: %d\n", total))
	buf.WriteString(fmt.Sprintf("convertSuccesses: %d (%d%%)\n", res.convertSuccesses, res.convertSuccesses*100/total))
	buf.WriteString(fmt.Sprintf("planSuccesses: %d (%d%%)\n", res.planSuccesses, res.planSuccesses*100/total))
	buf.WriteString(fmt.Sprintf("applySuccesses: %d (%d%%)\n", res.applySuccesses, res.applySuccesses*100/total))
	buf.WriteString(fmt.Sprintf("assertSuccesses: %d (%d%%)\n", res.assertSuccesses, res.assertSuccesses*100/res.assertTotal))
	return buf.String()
}

func runBenchmarkForLanguage(language string, skipTF bool, testCases []testCase) {
	switch language {
	case "typescript":
		tfResults := map[string]*benchmarkResult{}
		if !skipTF {
			tfResults = runTofuBenchmarks(testCases)
			fmt.Printf("tfResults:\n%s", formatResults(tfResults))
		}
		claudeResults := runPulumiBenchmarks(testCases, runClaudeConvert)
		fmt.Printf("claudeResults:\n%s", formatResults(claudeResults))
		pulumiResultsTs := runPulumiBenchmarks(testCases, runPulumiConvertTS)
		fmt.Printf("pulumiResultsTs:\n%s", formatResults(pulumiResultsTs))
		fmt.Println("--------------------------------")
		if !skipTF {
			fmt.Printf("tfResults:\n%s", resultSummary(tfResults))
		}
		fmt.Printf("claudeResults:\n%s", resultSummary(claudeResults))
		fmt.Printf("pulumiResultsTs:\n%s", resultSummary(pulumiResultsTs))
	default:
		// TODO: add other languages
		fmt.Printf("Language %s not supported\n", language)
		os.Exit(1)
	}
}

func runBenchmark(skipTF bool, testCases []testCase) {
	tfResults := map[string]*benchmarkResult{}
	if !skipTF {
		tfResults = runTofuBenchmarks(testCases)
		fmt.Printf("tfResults:\n%s", formatResults(tfResults))
	}

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
	if !skipTF {
		fmt.Printf("tfResults:\n%s", resultSummary(tfResults))
	}
	fmt.Printf("claudeResults:\n%s", resultSummary(claudeResults))
	fmt.Printf("pulumiResultsTs:\n%s", resultSummary(pulumiResultsTs))
	fmt.Printf("pulumiResultsPy:\n%s", resultSummary(pulumiResultsPy))
	fmt.Printf("pulumiResultsGo:\n%s", resultSummary(pulumiResultsGo))
	fmt.Printf("pulumiResultsCs:\n%s", resultSummary(pulumiResultsCs))
	fmt.Printf("pulumiResultsJava:\n%s", resultSummary(pulumiResultsJava))
	fmt.Printf("pulumiResultsYaml:\n%s", resultSummary(pulumiResultsYaml))
}

var allTestCases = map[string]testCase{
	"random_simple": {
		name: "random_simple",
		dir:  "programs/random_simple",
		assertions: map[string]assertion{
			"name is not empty": func(output map[string]any) error {
				if output["name"] == nil {
					return fmt.Errorf("name is nil")
				}
				if len(output["name"].(string)) == 0 {
					return fmt.Errorf("name is empty")
				}
				return nil
			},
		},
	},
	"aws_bucket": {
		name: "aws_bucket",
		dir:  "programs/aws_bucket",
		assertions: map[string]assertion{
			"s3 object content is correct": func(output map[string]any) error {
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
	},
	"aws_lambda_api": {
		name: "aws_lambda_api",
		dir:  "programs/aws_lambda_api",
		assertions: map[string]assertion{
			"lambda api response is correct": func(output map[string]any) error {
				time.Sleep(2 * time.Second)

				if output["url"] == nil {
					return fmt.Errorf("url is nil")
				}

				url := output["url"].(string)
				if len(url) == 0 {
					return fmt.Errorf("url is empty")
				}

				// make an http request to the url
				resp, err := http.Get(url + "/hello?Name=John")
				if err != nil {
					return fmt.Errorf("failed to make http request: %w", err)
				}
				defer resp.Body.Close()

				if resp.StatusCode != 200 {
					return fmt.Errorf("expected status code 200, got %d", resp.StatusCode)
				}

				body, err := io.ReadAll(resp.Body)
				if err != nil {
					return fmt.Errorf("failed to read response body: %w", err)
				}

				expected := `{"message":"Hello, John!"}`
				if string(body) != expected {
					return fmt.Errorf("expected %s, got %s", expected, string(body))
				}

				return nil
			},
		},
	},
}

func main() {
	language := flag.String("language", "typescript", "The language to benchmark. all will run all languages")
	skipTF := flag.Bool("skip-tf", false, "Skip the Terraform benchmark")
	example := flag.String("example", "all", "The example to run. all will run all examples")
	flag.Parse()

	testCases := []testCase{}
	if *example == "all" {
		for _, v := range allTestCases {
			testCases = append(testCases, v)
		}
	} else {
		tCase, ok := allTestCases[*example]
		if !ok {
			fmt.Printf("Example %s not found\n", *example)
			os.Exit(1)
		}
		testCases = []testCase{tCase}
	}

	if *language == "all" {
		runBenchmark(*skipTF, testCases)
	} else {
		runBenchmarkForLanguage(*language, *skipTF, testCases)
	}
}
