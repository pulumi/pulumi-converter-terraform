package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	mcp "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/stdio"
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
	prompt, err := readMcpPrompt("@pulumi/mcp-server@latest", "convert-terraform-to-typescript", outDir)
	if err != nil {
		return err
	}

	stdout, err := run(srcDir, "claude", "-p", prompt, "--add-dir", outDir, "--dangerously-skip-permissions")
	fmt.Printf("Claude convert stdout: %s\n", stdout)
	return err
}

func readMcpPrompt(mcpServer, promptName, outDir string) (string, error) {
	// Setup the stdio transport.
	cmd := exec.Command("npx", mcpServer, "stdio")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	if err := cmd.Start(); err != nil {
		return "", err
	}
	defer cmd.Process.Kill()
	transport := stdio.NewStdioServerTransportWithIO(stdout, stdin)
	client := mcp.NewClient(transport)
	if _, err := client.Initialize(context.Background()); err != nil {
		return "", err
	}

	// Prepare arguments for the prompt
	args := map[string]interface{}{
		"outputDir": outDir,
	}
	// Call the prompt
	resp, err := client.GetPrompt(context.Background(), promptName, args)
	if err != nil {
		return "", err
	}
	prompt := resp.Messages[0].Content.TextContent.Text
	return prompt, nil
}

type pulumiPlan struct {
	ResourcePlans map[string]struct {
		Steps []string `json:"steps"`
	} `json:"resourcePlans"`
}

func runPulumiPlan(dir string) (pulumiPlan, error) {
	_, err := runPulumi(dir, "stack", "init", "test")
	if err != nil {
		return pulumiPlan{}, err
	}
	_, err = runPulumi(dir, "preview", "--stack", "test", "--save-plan", "plan.out")
	if err != nil {
		return pulumiPlan{}, err
	}

	planFile, err := os.ReadFile(filepath.Join(dir, "plan.out"))
	if err != nil {
		return pulumiPlan{}, err
	}

	var p pulumiPlan
	err = json.Unmarshal(planFile, &p)
	if err != nil {
		return pulumiPlan{}, err
	}

	return p, nil
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

func getPulumiPlanResourceCount(plan pulumiPlan) int {
	count := 0
	for typ := range plan.ResourcePlans {
		if strings.Contains(typ, "::pulumi:pulumi:Stack::") {
			continue
		}
		if strings.Contains(typ, "::pulumi:providers:") {
			continue
		}
		count++
	}
	return count
}

func comparePulumiPlan(plan pulumiPlan, convertName, name string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	pulumiPlanFileName := filepath.Join(cwd, "plans", name, fmt.Sprintf("pulumi_plan_%s.out.json", convertName))

	err = saveOrCompareFile(pulumiPlanFileName, plan)
	if err != nil {
		return err
	}

	tfPlanFile, err := os.ReadFile(filepath.Join(cwd, "plans", name, "tf_plan.out.json"))
	if err != nil {
		return err
	}

	var tfPlanStruct tfPlan
	err = json.Unmarshal(tfPlanFile, &tfPlanStruct)
	if err != nil {
		return err
	}

	pulumiNumChanges := getPulumiPlanResourceCount(plan)
	tfNumChanges := len(tfPlanStruct.ResourceChanges)

	if pulumiNumChanges != tfNumChanges {
		return fmt.Errorf("pulumi num changes (%d) != tf num changes (%d)", pulumiNumChanges, tfNumChanges)
	}

	// TODO: compare resource types?

	return nil
}

func runPulumiBenchmarks(testCases []testCase, name string, convertFunc func(srcDir, outDir string) error) map[string]*benchmarkResult {
	results := map[string]*benchmarkResult{}
	for _, tc := range testCases {
		results[tc.name] = &benchmarkResult{
			assertSuccesses: map[string]bool{},
			planOnly:        tc.planOnly,
		}
		for k := range tc.assertions {
			results[tc.name].assertSuccesses[k] = false
		}

		dir, err := os.MkdirTemp("", "pulumi-benchmark")
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("dir: %s", dir)
		err = copyDirExcept(tc.dir, dir, ".tf")
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
			pulumiPlan, err := runPulumiPlan(dir)
			if err != nil {
				log.Printf("plan failed: %v", err)
				continue
			}
			results[tc.name].planSuccess = true

			err = comparePulumiPlan(pulumiPlan, name, tc.name)
			if err != nil {
				log.Printf("compare plan failed: %v", err)
				continue
			}
			results[tc.name].planComparisonSuccess = true
		}

		if tc.planOnly {
			continue
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
