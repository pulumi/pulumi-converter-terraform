package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

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
