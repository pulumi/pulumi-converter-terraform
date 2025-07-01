package main

import (
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
				out, err := getS3Object(output["url"].(string))
				if err != nil {
					return fmt.Errorf("failed to get s3 object: %w", err)
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
					return fmt.Errorf("url is nil")
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
					return fmt.Errorf("arn is nil")
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
					return fmt.Errorf("vpc is nil")
				}
				vpcId := output["vpc"].(string)
				err := checkVpcExists(vpcId)
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
