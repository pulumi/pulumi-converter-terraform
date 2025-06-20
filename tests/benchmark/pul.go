package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

func runPulumi(dir string, args ...string) ([]byte, error) {
	return run(dir, append([]string{"pulumi"}, args...)...)
}

func runPulumiConvert(srcDir string, outDir string, language string) error {
	_, err := runPulumi(srcDir, "convert", "--from", "terraform", "--language", language, "--out", outDir)
	if err != nil {
		return err
	}
	return nil
}

func runPulumiConvertTS(srcDir string, outDir string) error {
	return runPulumiConvert(srcDir, outDir, "typescript")
}

func runPulumiConvertPy(srcDir string, outDir string) error {
	return runPulumiConvert(srcDir, outDir, "python")
}

func runPulumiConvertGo(srcDir string, outDir string) error {
	return runPulumiConvert(srcDir, outDir, "go")
}

func runPulumiConvertCs(srcDir string, outDir string) error {
	return runPulumiConvert(srcDir, outDir, "csharp")
}

func runPulumiConvertJava(srcDir string, outDir string) error {
	return runPulumiConvert(srcDir, outDir, "java")
}

func runPulumiConvertYaml(srcDir string, outDir string) error {
	return runPulumiConvert(srcDir, outDir, "yaml")
}

func runClaudeConvert(srcDir string, outDir string) error {
	// This prompt is intentionally simplistic for now. We'll evolve it with larger test cases.
	prompt := "Convert this Terraform project to Pulumi TypeScript. Emit a full Pulumi project including package.json, tsconfig.json, and Pulumi.yaml."
	stdout, err := run(outDir, "claude", "-p", prompt, "--dangerously-skip-permissions")
	fmt.Printf("Claude convert stdout: %s\n", stdout)
	if err != nil {
		return err
	}

	_, err = run(outDir, "pulumi", "install")
	return err
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

func runPulumiDestroy(dir string) error {
	_, err := runPulumi(dir, "destroy", "--yes", "--stack", "test")
	if err != nil {
		return err
	}

	_, err = runPulumi(dir, "stack", "rm", "--force", "--stack", "test", "--yes")
	if err != nil {
		return err
	}

	return nil
}

func runPulumiBenchmarks(testCases []testCase, convertFunc func(srcDir, outDir string) error) map[string]*benchmarkResult {
	results := map[string]*benchmarkResult{}
	for _, tc := range testCases {
		results[tc.name] = &benchmarkResult{}
		dir, err := os.MkdirTemp("", "pulumi-benchmark")
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("dir: %s", dir)
		err = os.CopyFS(dir, os.DirFS(tc.dir))
		if err != nil {
			log.Fatal(err)
		}

		{
			err = convertFunc(tc.dir, dir)
			if err != nil {
				log.Printf("convert failed: %v", err)
				continue
			}
			results[tc.name].convertSuccess = true
		}

		defer runPulumiDestroy(dir)
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
