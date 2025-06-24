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
