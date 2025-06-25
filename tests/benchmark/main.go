package main

import (
	"encoding/json"
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
			"tags are correct": func(output map[string]any) error {
				if output["name"] == nil {
					return fmt.Errorf("name is nil")
				}

				name := output["name"].(string)
				if len(name) == 0 {
					return fmt.Errorf("name is empty")
				}

				out, err := run(".", "aws", "s3api", "get-bucket-tagging", "--bucket", name)
				if err != nil {
					return fmt.Errorf("failed to get bucket tags: %w", err)
				}

				type response struct {
					TagSet []struct {
						Key   string `json:"Key"`
						Value string `json:"Value"`
					} `json:"TagSet"`
				}

				var tags response
				err = json.Unmarshal(out, &tags)
				if err != nil {
					return fmt.Errorf("failed to unmarshal tags: %w", err)
				}

				tagsMap := make(map[string]string)
				for _, tag := range tags.TagSet {
					tagsMap[tag.Key] = tag.Value
				}

				if tagsMap["Name"] != "My bucket" {
					return fmt.Errorf("wrong tags: %v", tagsMap)
				}

				if tagsMap["my_tag"] != "my_value" {
					return fmt.Errorf("wrong tags: %v", tagsMap)
				}

				return nil
			},
		},
	},
	// adapted from https://github.com/hashicorp-education/learn-terraform-lambda-api-gateway
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
			"tags are correct": func(output map[string]any) error {
				if output["arn"] == nil {
					return fmt.Errorf("arn is nil")
				}
				arn := output["arn"].(string)
				out, err := run(".", "aws", "lambda", "list-tags", "--resource", arn)
				if err != nil {
					return fmt.Errorf("failed to list tags: %w", err)
				}

				type response struct {
					Tags map[string]string `json:"Tags"`
				}

				var tags response
				err = json.Unmarshal(out, &tags)
				if err != nil {
					return fmt.Errorf("failed to unmarshal tags: %w", err)
				}

				if tags.Tags["project"] != "aws_lambda_api" {
					return fmt.Errorf("wrong tags: %v", tags)
				}

				if tags.Tags["environment"] != "test" {
					return fmt.Errorf("wrong tags: %v", tags)
				}

				if tags.Tags["my_tag"] != "my_value" {
					return fmt.Errorf("wrong tags: %v", tags)
				}

				return nil
			},
		},
	},
	// adapted from https://github.com/hashicorp-education/learn-terraform-cloudflare-static-website
	"cloudflare_aws_static_website": {
		name:     "cloudflare_aws_static_website",
		dir:      "programs/cloudflare_aws_static_website",
		planOnly: true,
	},
}

func main() {
	language := flag.String("language", "typescript", "The language to benchmark. all will run all languages")
	skipTF := flag.Bool("skip-tf", false, "Skip the Terraform benchmark")
	skipLLM := flag.Bool("skip-llm", false, "Skip the LLM benchmark")
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

	opts := benchmarkOptions{
		skipTF:  *skipTF,
		skipLLM: *skipLLM,
	}

	if *language == "all" {
		runBenchmark(opts, testCases)
	} else {
		runBenchmarkForLanguage(*language, opts, testCases)
	}
}
