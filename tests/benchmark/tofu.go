package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

func runTF(dir string, args ...string) ([]byte, error) {
	return run(dir, append([]string{"tofu"}, args...)...)
}

type tfOutput struct {
	Value any `json:"value"`
}

type tfPlan struct {
	ResourceChanges []struct {
		Type string `json:"type"`
		Name string `json:"name"`
	} `json:"resource_changes"`
	OutputChanges map[string]struct {
		Actions []string `json:"actions"`
	} `json:"output_changes"`
}

func runTFPlan(dir string) (tfPlan, error) {
	_, err := runTF(dir, "init")
	if err != nil {
		return tfPlan{}, err
	}
	_, err = runTF(dir, "plan", "-out", "plan.out")
	if err != nil {
		return tfPlan{}, err
	}

	stdout, err := runTF(dir, "show", "-json", "plan.out")
	if err != nil {
		return tfPlan{}, err
	}

	var plan tfPlan
	err = json.Unmarshal(stdout, &plan)
	if err != nil {
		return tfPlan{}, err
	}

	return plan, nil
}

func runTFApply(dir string) (map[string]any, error) {
	_, err := runTF(dir, "apply", "-auto-approve")
	if err != nil {
		return nil, err
	}

	tfOutput := map[string]tfOutput{}
	stdout, err := runTF(dir, "output", "-json")
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

func runTFDestroy(dir string) error {
	_, err := runTF(dir, "destroy", "-auto-approve")
	if err != nil {
		return err
	}
	return nil
}

type assertion func(output map[string]any) error

type testCase struct {
	name       string
	dir        string
	planOnly   bool
	assertions map[string]assertion
}

type benchmarkResult struct {
	convertSuccess        bool
	planSuccess           bool
	planComparisonSuccess bool
	planOnly              bool
	applySuccess          bool
	assertSuccesses       map[string]bool
}

func recordTFPlan(name string, plan tfPlan) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	path := filepath.Join(cwd, "plans", name, "tf_plan.out.json")

	err = saveOrCompareFile(path, plan)
	return err
}

func runTFBenchmarks(testCases []testCase) map[string]*benchmarkResult {
	results := map[string]*benchmarkResult{}
	for _, tc := range testCases {
		results[tc.name] = &benchmarkResult{
			assertSuccesses: map[string]bool{},
			planOnly:        tc.planOnly,
		}
		for k := range tc.assertions {
			results[tc.name].assertSuccesses[k] = false
		}

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
			plan, err := runTFPlan(dir)
			if err != nil {
				log.Printf("plan failed: %v", err)
				continue
			}
			results[tc.name].planSuccess = true
			err = recordTFPlan(tc.name, plan)
			if err != nil {
				log.Printf("plan comparison failed: %v", err)
			} else {
				results[tc.name].planComparisonSuccess = true
			}
		}

		if tc.planOnly {
			continue
		}

		defer runTFDestroy(dir)
		output := map[string]any{}

		{
			output, err = runTFApply(dir)
			if err != nil {
				log.Printf("apply failed: %v", err)
				continue
			}
			results[tc.name].applySuccess = true
		}

		{
			for k, assertion := range tc.assertions {
				err = assertion(output)
				if err != nil {
					results[tc.name].assertSuccesses[k] = false
					continue
				}
				results[tc.name].assertSuccesses[k] = true
			}
		}
	}
	return results
}
