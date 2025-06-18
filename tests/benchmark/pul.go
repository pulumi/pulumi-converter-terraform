package main

import (
	"encoding/json"
	"log"
	"os"
)

func runPulumi(dir string, args ...string) ([]byte, error) {
	return run(dir, append([]string{"pulumi"}, args...)...)
}

func runPulumiConvert(srcDir string, outDir string) error {
	_, err := runPulumi(srcDir, "convert", "--from", "terraform", "--language", "typescript", "--out", outDir)
	if err != nil {
		return err
	}
	return nil
}

func runPulumiPlan(dir string) error {
	_, err := runPulumi(dir, "stack", "init", "test")
	if err != nil {
		return err
	}
	_, err = runPulumi(dir, "preview", "--stack", "test")
	if err != nil {
		return err
	}
	return nil
}

func runPulumiApply(dir string) (map[string]any, error) {
	defer runPulumi(dir, "stack", "rm", "--force", "--stack", "test", "--yes")
	defer runPulumi(dir, "destroy", "--yes", "--stack", "test")
	_, err := runPulumi(dir, "up", "--yes", "--stack", "test")
	if err != nil {
		return nil, err
	}

	output := map[string]any{}
	stdout, err := runPulumi(dir, "stack", "output", "-json", "--stack", "test")
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(stdout, &output)
	if err != nil {
		return nil, err
	}

	return output, nil
}

func runPulumiBenchmarks(testCases []testCase) map[string]*benchmarkResult {
	results := map[string]*benchmarkResult{}
	for _, tc := range testCases {
		results[tc.name] = &benchmarkResult{}
		dir, err := os.MkdirTemp("", "pulumi-benchmark")
		if err != nil {
			log.Fatal(err)
		}
		err = os.CopyFS(dir, os.DirFS(tc.dir))
		if err != nil {
			log.Fatal(err)
		}

		{
			err = runPulumiConvert(tc.dir, dir)
			if err != nil {
				log.Printf("convert failed: %v", err)
				continue
			}
			results[tc.name].convertSuccess = true
		}

		{
			err = runPulumiPlan(dir)
			if err != nil {
				log.Printf("plan failed: %v", err)
				continue
			}
			results[tc.name].planSuccess = true
		}

		output := map[string]any{}

		{
			output, err = runPulumiApply(dir)
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
