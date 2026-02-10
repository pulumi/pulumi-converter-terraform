// Copyright 2026, Pulumi Corporation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"errors"
	"flag"
	"fmt"
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
					return errors.New("name is nil")
				}
				if len(output["name"].(string)) == 0 {
					return errors.New("name is empty")
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
					return errors.New("url is nil")
				}
				out, err := getS3Object(output["url"].(string))
				if err != nil {
					return fmt.Errorf("failed to get s3 object: %w", err)
				}
				if out != "hi" {
					return fmt.Errorf("expected 'hi', got %s", out)
				}

				return nil
			},
			"tags are correct": func(output map[string]any) error {
				if output["name"] == nil {
					return errors.New("name is nil")
				}

				name := output["name"].(string)
				tagsMap, err := getS3BucketTags(name)
				if err != nil {
					return fmt.Errorf("failed to get bucket tags: %w", err)
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
					return errors.New("url is nil")
				}

				url := output["url"].(string)
				message, err := callLambda(url + "/hello?Name=John")
				if err != nil {
					return fmt.Errorf("failed to call lambda: %w", err)
				}

				if message != "Hello, John!" {
					return fmt.Errorf("expected 'Hello, John!', got %s", message)
				}
				return nil
			},
			"tags are correct": func(output map[string]any) error {
				if output["arn"] == nil {
					return errors.New("arn is nil")
				}
				arn := output["arn"].(string)
				tags, err := getLambdaTags(arn)
				if err != nil {
					return fmt.Errorf("failed to get lambda tags: %w", err)
				}

				if tags["project"] != "aws_lambda_api" {
					return fmt.Errorf("wrong tags: %v", tags)
				}

				if tags["environment"] != "test" {
					return fmt.Errorf("wrong tags: %v", tags)
				}

				if tags["my_tag"] != "my_value" {
					return fmt.Errorf("wrong tags: %v", tags)
				}

				return nil
			},
		},
	},
	// adapted from https://github.com/corymhall/example-terraform-project
	"aws_vpc": {
		name: "aws_vpc",
		dir:  "programs/aws_vpc",
		assertions: map[string]assertion{
			"vpc exists": func(output map[string]any) error {
				if output["vpc"] == nil {
					return errors.New("vpc is nil")
				}
				vpcID := output["vpc"].(string)
				err := checkVpcExists(vpcID)
				if err != nil {
					return fmt.Errorf("vpc does not exist: %w", err)
				}
				return nil
			},
		},
	},
	// adapted from https://github.com/corymhall/example-terraform-project
	"aws_project": {
		name:     "aws_project",
		dir:      "programs/aws_project",
		planOnly: true,
	},
	// adapted from https://github.com/hashicorp-education/learn-terraform-cloudflare-static-website
	"cloudflare_aws_static_website": {
		name:     "cloudflare_aws_static_website",
		dir:      "programs/cloudflare_aws_static_website",
		planOnly: true,
	},
	// adapted from https://github.com/hashicorp-education/learn-terraform-provision-eks-cluster
	"aws_eks_cluster": {
		name:     "aws_eks_cluster",
		dir:      "programs/aws_eks_cluster",
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
