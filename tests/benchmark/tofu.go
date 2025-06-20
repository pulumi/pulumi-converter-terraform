package main

import (
	"encoding/json"
	"log"
	"os"
)

func runTofu(dir string, args ...string) ([]byte, error) {
	return run(dir, append([]string{"tofu"}, args...)...)
}

type tfOutput struct {
	Value any `json:"value"`
}

func runTofuPlan(dir string) error {
	_, err := runTofu(dir, "init")
	if err != nil {
		return err
	}
	_, err = runTofu(dir, "plan")
	if err != nil {
		return err
	}
	return nil
}

func runTofuApply(dir string) (map[string]any, error) {
	_, err := runTofu(dir, "apply", "-auto-approve")
	if err != nil {
		return nil, err
	}

	tfOutput := map[string]tfOutput{}
	stdout, err := runTofu(dir, "output", "-json")
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(stdout, &tfOutput)
	if err != nil {
		return nil, err
	}

	output := map[string]any{}
	for k, v := range tfOutput {
		output[k] = v.Value
	}
	return output, nil
}

func runTofuDestroy(dir string) error {
	_, err := runTofu(dir, "destroy", "-auto-approve")
	if err != nil {
		return err
	}
	return nil
}

type testCase struct {
	name      string
	dir       string
	assertion func(output map[string]any) error
}

type benchmarkResult struct {
	convertSuccess bool
	planSuccess    bool
	applySuccess   bool
	assertSuccess  bool
}

func runTofuBenchmarks(testCases []testCase) map[string]*benchmarkResult {
	results := map[string]*benchmarkResult{}
	for _, tc := range testCases {
		results[tc.name] = &benchmarkResult{}
		dir, err := os.MkdirTemp("", "tofu-benchmark")
		if err != nil {
			log.Fatal(err)
		}
		err = os.CopyFS(dir, os.DirFS(tc.dir))
		if err != nil {
			log.Fatal(err)
		}

		results[tc.name].convertSuccess = true

		{
			err = runTofuPlan(dir)
			if err != nil {
				log.Printf("plan failed: %v", err)
				continue
			}
			results[tc.name].planSuccess = true
		}

		defer runTofuDestroy(dir)
		output := map[string]any{}

		{
			output, err = runTofuApply(dir)
			if err != nil {
				log.Printf("apply failed: %v", err)
				continue
			}
			results[tc.name].applySuccess = true
		}

		{
			err = tc.assertion(output)
			if err != nil {
				log.Printf("assertion failed: %v", err)
				continue
			}
			results[tc.name].assertSuccess = true
		}
	}
	return results
}
