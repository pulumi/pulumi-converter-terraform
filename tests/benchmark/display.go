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
	"bytes"
	"fmt"
	"os"
)

func formatResults(results map[string]*benchmarkResult) string {
	buf := bytes.Buffer{}
	for k, v := range results {
		buf.WriteString(fmt.Sprintf("%s: %+v\n", k, v))
	}
	return buf.String()
}

func getPercentage(numerator, denominator int) int {
	if denominator == 0 {
		return 100
	}

	return numerator * 100 / denominator
}

func resultSummary(results map[string]*benchmarkResult) string {
	buf := bytes.Buffer{}
	total := len(results)
	type summary struct {
		convertSuccesses        int
		planSuccesses           int
		planComparisonSuccesses int
		applySuccesses          int
		assertSuccesses         int
		applyTotal              int
		assertTotal             int
	}
	res := summary{}
	for _, v := range results {
		if v.convertSuccess {
			res.convertSuccesses++
		}
		if v.planSuccess {
			res.planSuccesses++
		}
		if v.planComparisonSuccess {
			res.planComparisonSuccesses++
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
		if !v.planOnly {
			res.applyTotal++
		}
	}

	buf.WriteString(fmt.Sprintf("total: %d\n", total))
	buf.WriteString(fmt.Sprintf(
		"convertSuccesses: %d (%d%%)\n",
		res.convertSuccesses,
		getPercentage(res.convertSuccesses, total)))
	buf.WriteString(fmt.Sprintf(
		"planSuccesses: %d (%d%%)\n",
		res.planSuccesses,
		getPercentage(res.planSuccesses, total)))
	buf.WriteString(fmt.Sprintf(
		"planComparisonSuccesses: %d (%d%%)\n",
		res.planComparisonSuccesses,
		getPercentage(res.planComparisonSuccesses, total)))
	buf.WriteString(fmt.Sprintf(
		"applySuccesses: %d (%d%%)\n",
		res.applySuccesses,
		getPercentage(res.applySuccesses, res.applyTotal)))
	buf.WriteString(fmt.Sprintf(
		"assertSuccesses: %d (%d%%)\n",
		res.assertSuccesses,
		getPercentage(res.assertSuccesses, res.assertTotal)))
	return buf.String()
}

type benchmarkOptions struct {
	skipTF  bool
	skipLLM bool
}

func runBenchmarkForLanguage(language string, opts benchmarkOptions, testCases []testCase) {
	switch language {
	case "typescript":
		tfResults := map[string]*benchmarkResult{}
		if !opts.skipTF {
			tfResults = runTFBenchmarks(testCases)
			fmt.Printf("tfResults:\n%s", formatResults(tfResults))
		}
		claudeResults := map[string]*benchmarkResult{}
		if !opts.skipLLM {
			claudeResults = runPulumiBenchmarks(testCases, "claude", runClaudeConvert)
			fmt.Printf("claudeResults:\n%s", formatResults(claudeResults))
		}
		pulumiResultsTs := runPulumiBenchmarks(testCases, "converter-ts", runPulumiConvertTS)
		fmt.Printf("pulumiResultsTs:\n%s", formatResults(pulumiResultsTs))
		fmt.Println("--------------------------------")
		if !opts.skipTF {
			fmt.Printf("tfResults:\n%s", resultSummary(tfResults))
		}
		if !opts.skipLLM {
			fmt.Printf("claudeResults:\n%s", resultSummary(claudeResults))
		}
		fmt.Printf("pulumiResultsTs:\n%s", resultSummary(pulumiResultsTs))
	default:
		// TODO: add other languages
		fmt.Printf("Language %s not supported\n", language)
		os.Exit(1)
	}
}

func runBenchmark(opts benchmarkOptions, testCases []testCase) {
	tfResults := map[string]*benchmarkResult{}
	if !opts.skipTF {
		tfResults = runTFBenchmarks(testCases)
		fmt.Printf("tfResults:\n%s", formatResults(tfResults))
	}

	claudeResults := map[string]*benchmarkResult{}
	if !opts.skipLLM {
		claudeResults = runPulumiBenchmarks(testCases, "claude", runClaudeConvert)
		fmt.Printf("claudeResults:\n%s", formatResults(claudeResults))
	}
	pulumiResultsTs := runPulumiBenchmarks(testCases, "converter-ts", runPulumiConvertTS)
	fmt.Printf("pulumiResultsTs:\n%s", formatResults(pulumiResultsTs))
	pulumiResultsPy := runPulumiBenchmarks(testCases, "converter-py", runPulumiConvertPy)
	fmt.Printf("pulumiResultsPy:\n%s", formatResults(pulumiResultsPy))
	pulumiResultsGo := runPulumiBenchmarks(testCases, "converter-go", runPulumiConvertGo)
	fmt.Printf("pulumiResultsGo:\n%s", formatResults(pulumiResultsGo))
	pulumiResultsCs := runPulumiBenchmarks(testCases, "converter-cs", runPulumiConvertCs)
	fmt.Printf("pulumiResultsCs:\n%s", formatResults(pulumiResultsCs))
	pulumiResultsJava := runPulumiBenchmarks(testCases, "converter-java", runPulumiConvertJava)
	fmt.Printf("pulumiResultsJava:\n%s", formatResults(pulumiResultsJava))
	pulumiResultsYaml := runPulumiBenchmarks(testCases, "converter-yaml", runPulumiConvertYaml)
	fmt.Printf("pulumiResultsYaml:\n%s", formatResults(pulumiResultsYaml))
	fmt.Println("--------------------------------")
	if !opts.skipTF {
		fmt.Printf("tfResults:\n%s", resultSummary(tfResults))
	}
	if !opts.skipLLM {
		fmt.Printf("claudeResults:\n%s", resultSummary(claudeResults))
	}
	fmt.Printf("pulumiResultsTs:\n%s", resultSummary(pulumiResultsTs))
	fmt.Printf("pulumiResultsPy:\n%s", resultSummary(pulumiResultsPy))
	fmt.Printf("pulumiResultsGo:\n%s", resultSummary(pulumiResultsGo))
	fmt.Printf("pulumiResultsCs:\n%s", resultSummary(pulumiResultsCs))
	fmt.Printf("pulumiResultsJava:\n%s", resultSummary(pulumiResultsJava))
	fmt.Printf("pulumiResultsYaml:\n%s", resultSummary(pulumiResultsYaml))
}
