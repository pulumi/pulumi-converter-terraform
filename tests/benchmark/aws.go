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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func getS3Object(url string) (string, error) {
	if !strings.HasPrefix(url, "s3://") {
		return "", fmt.Errorf("url is not an s3 url: %s", url)
	}
	out, err := run(".", "aws", "s3", "cp", url, "-")
	if err != nil {
		return "", fmt.Errorf("failed to copy object: %w", err)
	}
	return string(out), nil
}

func getS3BucketTags(bucket string) (map[string]string, error) {
	if bucket == "" {
		return nil, errors.New("bucket name is empty")
	}

	out, err := run(".", "aws", "s3api", "get-bucket-tagging", "--bucket", bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket tags: %w", err)
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
		return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
	}

	tagsMap := make(map[string]string)
	for _, tag := range tags.TagSet {
		tagsMap[tag.Key] = tag.Value
	}

	return tagsMap, nil
}

func callLambda(url string) (string, error) {
	if url == "" {
		return "", errors.New("url is empty")
	}

	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		return "", fmt.Errorf("failed to make http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("expected status code 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	type response struct {
		Message string `json:"message"`
	}

	var respBody response
	err = json.Unmarshal(body, &respBody)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return respBody.Message, nil
}

func getLambdaTags(arn string) (map[string]string, error) {
	if arn == "" {
		return nil, errors.New("arn is empty")
	}

	out, err := run(".", "aws", "lambda", "list-tags", "--resource", arn)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}

	type response struct {
		Tags map[string]string `json:"Tags"`
	}

	var tags response
	err = json.Unmarshal(out, &tags)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
	}

	return tags.Tags, nil
}

func checkVpcExists(vpcID string) error {
	if vpcID == "" {
		return errors.New("vpc id is empty")
	}

	_, err := run(".", "aws", "ec2", "describe-vpcs", "--filters", "Name=vpc-id,Values="+vpcID)
	if err != nil {
		return fmt.Errorf("failed to describe vpc: %w", err)
	}
	return nil
}
