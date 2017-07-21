package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-tools/go-steputils/input"
)

// ConfigsModel ...
type ConfigsModel struct {
	// api
	APIBaseURL string
	BuildSlug  string
	AppSlug    string

	// shared
	ApkPath      string
	TestApkPath  string
	TestType     string
	TestDevices  string
	AppPackageID string
	TestTimeout  string

	// instrumentation
	InstTestPackageID   string
	InstTestRunnerClass string
	InstTestTargets     string

	// robo
	RoboInitialActivity string
	RoboMaxDepth        string
	RoboMaxSteps        string
	RoboDirectives      string

	// loop
	LoopScenarios      string
	LoopScenarioLabels string
}

type ListStepsResponse struct {
	Steps []*Step `json:"steps,omitempty"`
}

type Step struct {
	State string `json:"state,omitempty"`
}

type AndroidDevice struct {
	AndroidModelId   string `json:"androidModelId,omitempty"`
	AndroidVersionId string `json:"androidVersionId,omitempty"`
	Locale           string `json:"locale,omitempty"`
	Orientation      string `json:"orientation,omitempty"`
}

type AndroidDeviceList struct {
	AndroidDevices []*AndroidDevice `json:"androidDevices,omitempty"`
}

type EnvironmentMatrix struct {
	AndroidDeviceList *AndroidDeviceList `json:"androidDeviceList,omitempty"`
}

type TestMatrix struct {
	EnvironmentMatrix *EnvironmentMatrix `json:"environmentMatrix,omitempty"`
	TestSpecification *TestSpecification `json:"testSpecification,omitempty"`
}

type TestSpecification struct {
	AndroidInstrumentationTest *AndroidInstrumentationTest `json:"androidInstrumentationTest,omitempty"`
	AndroidRoboTest            *AndroidRoboTest            `json:"androidRoboTest,omitempty"`
	AndroidTestLoop            *AndroidTestLoop            `json:"androidTestLoop,omitempty"`
	AutoGoogleLogin            bool                        `json:"autoGoogleLogin,omitempty"`
	TestSetup                  *TestSetup                  `json:"testSetup,omitempty"`
	TestTimeout                string                      `json:"testTimeout,omitempty"`
}

type AndroidInstrumentationTest struct {
	AppPackageId    string   `json:"appPackageId,omitempty"`
	TestPackageId   string   `json:"testPackageId,omitempty"`
	TestRunnerClass string   `json:"testRunnerClass,omitempty"`
	TestTargets     []string `json:"testTargets,omitempty"`
}

type AndroidRoboTest struct {
	AppInitialActivity string           `json:"appInitialActivity,omitempty"`
	AppPackageId       string           `json:"appPackageId,omitempty"`
	MaxDepth           int64            `json:"maxDepth,omitempty"`
	MaxSteps           int64            `json:"maxSteps,omitempty"`
	RoboDirectives     []*RoboDirective `json:"roboDirectives,omitempty"`
}

type RoboDirective struct {
	ActionType   string `json:"actionType,omitempty"`
	InputText    string `json:"inputText,omitempty"`
	ResourceName string `json:"resourceName,omitempty"`
}

type AndroidTestLoop struct {
	AppPackageId   string   `json:"appPackageId,omitempty"`
	ScenarioLabels []string `json:"scenarioLabels,omitempty"`
	Scenarios      []int64  `json:"scenarios,omitempty"`
}

type TestSetup struct {
	DirectoriesToPull    []string               `json:"directoriesToPull,omitempty"`
	EnvironmentVariables []*EnvironmentVariable `json:"environmentVariables,omitempty"`
	NetworkProfile       string                 `json:"networkProfile,omitempty"`
}

type EnvironmentVariable struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

// TestExecutionsModel ...
type TestExecutionsModel []struct {
	State       string `json:"state"`
	TestDetails struct {
		ProgressMessages []string `json:"progressMessages"`
	} `json:"testDetails"`
	ToolResultStep struct {
		ExecutionID string `json:"executionId"`
	} `json:"toolResultsStep"`
}

// MatrixSummary ...
type MatrixSummary struct {
	ExecutionID string `json:"executionId"`
	State       string `json:"state"`
	Outcome     struct {
		Summary string `json:"summary"`
	} `json:"outcome"`
}

// UploadURLRequest ...
type UploadURLRequest struct {
	AppURL     string `json:"appUrl"`
	TestAppURL string `json:"testAppUrl"`
}

func createConfigsModelFromEnvs() ConfigsModel {
	return ConfigsModel{
		// api
		APIBaseURL: os.Getenv("api_base_url"),
		BuildSlug:  os.Getenv("BITRISE_BUILD_SLUG"),
		AppSlug:    os.Getenv("BITRISE_APP_SLUG"),

		// shared
		ApkPath:      os.Getenv("apk_path"),
		TestApkPath:  os.Getenv("test_apk_path"),
		TestType:     os.Getenv("test_type"),
		TestDevices:  os.Getenv("test_devices"),
		AppPackageID: os.Getenv("app_package_id"),
		TestTimeout:  os.Getenv("test_timeout"),

		// instrumentation
		InstTestPackageID:   os.Getenv("inst_test_package_id"),
		InstTestRunnerClass: os.Getenv("inst_test_runner_class"),
		InstTestTargets:     os.Getenv("inst_test_targets"),

		// robo
		RoboInitialActivity: os.Getenv("robo_initial_activity"),
		RoboMaxDepth:        os.Getenv("robo_max_depth"),
		RoboMaxSteps:        os.Getenv("robo_max_steps"),
		RoboDirectives:      os.Getenv("robo_directives"),

		// loop
		LoopScenarios:      os.Getenv("loop_scenarios"),
		LoopScenarioLabels: os.Getenv("loop_scenario_labels"),
	}
}

func (configs ConfigsModel) print() {
	log.Infof("Configs:")
	log.Printf("- APIBaseURL: %s", configs.APIBaseURL)
	log.Printf("- BuildSlug: %s", configs.BuildSlug)
	log.Printf("- AppSlug: %s", configs.AppSlug)
	log.Printf("- ApkPath: %s", configs.ApkPath)
	log.Printf("- TestApkPath: %s", configs.TestApkPath)
	log.Printf("- TestType: %s", configs.TestType)
	log.Printf("- AppPackageID: %s", configs.AppPackageID)
	log.Printf("- TestTimeout: %s", configs.TestTimeout)
	log.Printf("- TestDevices:\n%s", configs.TestDevices)
}

func (configs ConfigsModel) validate() error {

	if err := input.ValidateIfNotEmpty(configs.APIBaseURL); err != nil {
		return fmt.Errorf("Issue with APIBaseURL: %s", err)
	}
	if err := input.ValidateIfNotEmpty(configs.BuildSlug); err != nil {
		return fmt.Errorf("Issue with BuildSlug: %s", err)
	}
	if err := input.ValidateIfNotEmpty(configs.AppSlug); err != nil {
		return fmt.Errorf("Issue with AppSlug: %s", err)
	}
	if err := input.ValidateIfNotEmpty(configs.TestType); err != nil {
		return fmt.Errorf("Issue with TestType: %s", err)
	}
	if err := input.ValidateWithOptions(configs.TestType, "instrumentation", "robo", "gameloop"); err != nil {
		return fmt.Errorf("Issue with TestType: %s", err)
	}
	if err := input.ValidateIfNotEmpty(configs.ApkPath); err != nil {
		return fmt.Errorf("Issue with ApkPath: %s", err)
	}
	if err := input.ValidateIfPathExists(configs.ApkPath); err != nil {
		return fmt.Errorf("Issue with ApkPath: %s", err)
	}
	if configs.TestType == "instrumentation" {
		if err := input.ValidateIfNotEmpty(configs.TestApkPath); err != nil {
			return fmt.Errorf("Issue with TestApkPath: %s", err)
		}
		if err := input.ValidateIfPathExists(configs.TestApkPath); err != nil {
			return fmt.Errorf("Issue with TestApkPath: %s", err)
		}
	}

	return nil
}

func failf(f string, v ...interface{}) {
	log.Errorf(f, v)
	os.Exit(1)
}

func main() {
	configs := createConfigsModelFromEnvs()

	fmt.Println()
	configs.print()

	if err := configs.validate(); err != nil {
		failf("%s", err)
	}

	fmt.Println()

	log.Infof("Upload APKs")
	{
		url := configs.APIBaseURL + "/assets/" + configs.AppSlug + "/" + configs.BuildSlug

		req, err := http.NewRequest("POST", url, nil)
		if err != nil {
			failf("Failed to create http request, error: %s", err)
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			failf("Failed to get http response, error: %s", err)
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			failf("Failed to read response body, error: %s", err)
		}

		responseModel := &UploadURLRequest{}

		err = json.Unmarshal(body, responseModel)
		if err != nil {
			failf("Failed to unmarshal response body, error: %s", err)
		}

		err = _tryToUploadArchive(responseModel.AppURL, configs.ApkPath)
		if err != nil {
			failf("Failed to upload file(%s) to (%s), error: %s", configs.ApkPath, responseModel.AppURL, err)
		}
		err = _tryToUploadArchive(responseModel.TestAppURL, configs.TestApkPath)
		if err != nil {
			failf("Failed to upload file(%s) to (%s), error: %s", configs.TestApkPath, responseModel.TestAppURL, err)
		}

		log.Donef("=> APKs uploaded")
	}

	log.Infof("Start test")
	{
		url := configs.APIBaseURL + "/" + configs.AppSlug + "/" + configs.BuildSlug

		testModel := &TestMatrix{}
		testModel.EnvironmentMatrix = &EnvironmentMatrix{AndroidDeviceList: &AndroidDeviceList{}}
		testModel.EnvironmentMatrix.AndroidDeviceList.AndroidDevices = []*AndroidDevice{}

		scanner := bufio.NewScanner(strings.NewReader(configs.TestDevices))
		for scanner.Scan() {
			device := scanner.Text()
			device = strings.TrimSpace(device)
			if device == "" {
				continue
			}

			deviceParams := strings.Split(device, ",")
			if len(deviceParams) != 4 {
				failf("Invalid test device configuration: %s", device)
			}

			newDevice := AndroidDevice{
				AndroidModelId:   deviceParams[0],
				AndroidVersionId: deviceParams[1],
				Locale:           deviceParams[2],
				Orientation:      deviceParams[3],
			}

			testModel.EnvironmentMatrix.AndroidDeviceList.AndroidDevices = append(testModel.EnvironmentMatrix.AndroidDeviceList.AndroidDevices, &newDevice)
		}

		testModel.TestSpecification = &TestSpecification{
			TestTimeout: fmt.Sprintf("%ss", configs.TestTimeout),
		}

		switch configs.TestType {
		case "instrumentation":
			testModel.TestSpecification.AndroidInstrumentationTest = &AndroidInstrumentationTest{}
			if configs.AppPackageID != "" {
				testModel.TestSpecification.AndroidInstrumentationTest.AppPackageId = configs.AppPackageID
			}
			if configs.InstTestPackageID != "" {
				testModel.TestSpecification.AndroidInstrumentationTest.TestPackageId = configs.InstTestPackageID
			}
			if configs.InstTestRunnerClass != "" {
				testModel.TestSpecification.AndroidInstrumentationTest.TestRunnerClass = configs.InstTestRunnerClass
			}
			if configs.InstTestTargets != "" {
				targets := strings.Split(strings.TrimSpace(configs.InstTestTargets), ",")
				testModel.TestSpecification.AndroidInstrumentationTest.TestTargets = targets
			}
		case "robo":
			testModel.TestSpecification.AndroidRoboTest = &AndroidRoboTest{}
			if configs.AppPackageID != "" {
				testModel.TestSpecification.AndroidRoboTest.AppPackageId = configs.AppPackageID
			}
			if configs.RoboInitialActivity != "" {
				testModel.TestSpecification.AndroidRoboTest.AppInitialActivity = configs.RoboInitialActivity
			}
			if configs.RoboMaxDepth != "" {
				maxDepth, err := strconv.Atoi(configs.RoboMaxDepth)
				if err != nil {
					failf("Failed to parse string(%s) to integer, error: %s", configs.RoboMaxDepth, err)
				}
				testModel.TestSpecification.AndroidRoboTest.MaxDepth = int64(maxDepth)
			}
			if configs.RoboMaxSteps != "" {
				maxSteps, err := strconv.Atoi(configs.RoboMaxSteps)
				if err != nil {
					failf("Failed to parse string(%s) to integer, error: %s", configs.RoboMaxSteps, err)
				}
				testModel.TestSpecification.AndroidRoboTest.MaxSteps = int64(maxSteps)
			}
			if configs.RoboDirectives != "" {
				roboDirectives := []*RoboDirective{}
				scanner := bufio.NewScanner(strings.NewReader(configs.RoboDirectives))
				for scanner.Scan() {
					directive := scanner.Text()
					directive = strings.TrimSpace(directive)
					if directive == "" {
						continue
					}

					directiveParams := strings.Split(directive, ",")
					if len(directiveParams) != 3 {
						failf("Invalid directive configuration: %s", directive)
					}
					roboDirectives = append(roboDirectives, &RoboDirective{ResourceName: directiveParams[0], InputText: directiveParams[1], ActionType: directiveParams[2]})
				}
				testModel.TestSpecification.AndroidRoboTest.RoboDirectives = roboDirectives
			}
		case "gameloop":
			testModel.TestSpecification.AndroidTestLoop = &AndroidTestLoop{}
			if configs.AppPackageID != "" {
				testModel.TestSpecification.AndroidTestLoop.AppPackageId = configs.AppPackageID
			}
			if configs.LoopScenarios != "" {
				loopScenarios := []int64{}
				for _, scenarioStr := range strings.Split(strings.TrimSpace(configs.LoopScenarios), ",") {
					scenario, err := strconv.Atoi(scenarioStr)
					if err != nil {
						failf("Failed to parse string(%s) to integer, error: %s", scenarioStr, err)
					}
					loopScenarios = append(loopScenarios, int64(scenario))
				}
				testModel.TestSpecification.AndroidTestLoop.Scenarios = loopScenarios
			}
			if configs.LoopScenarioLabels != "" {
				scenarioLabels := strings.Split(strings.TrimSpace(configs.LoopScenarioLabels), ",")
				testModel.TestSpecification.AndroidTestLoop.ScenarioLabels = scenarioLabels
			}
		}

		jsonByte, err := json.Marshal(testModel)
		if err != nil {
			failf("Failed to marshal test model, error: %s", err)
		}

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonByte))
		if err != nil {
			failf("Failed to create http request, error: %s", err)
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			failf("Failed to get http response, error: %s", err)
		}

		if resp.StatusCode != http.StatusOK {
			failf("Failed to get http response, status code: %d", resp.StatusCode)
		}

		log.Donef("=> Test started")
	}

	log.Infof("Waiting for test results")
	{
		finished := false
		for !finished {
			time.Sleep(5 * time.Second)

			url := configs.APIBaseURL + "/" + configs.AppSlug + "/" + configs.BuildSlug

			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				failf("Failed to create http request, error: %s", err)
			}

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				failf("Failed to get http response, error: %s", err)
			}

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				failf("Failed to read response body, error: %s", err)
			}

			log.Warnf("%s", body)

			responseModel := &ListStepsResponse{}

			err = json.Unmarshal(body, responseModel)
			if err != nil {
				failf("Failed to unmarshal response body, error: %s", err)
			}

			finished = true
			for _, step := range responseModel.Steps {
				if step.State != "complete" {
					finished = false
				}
			}

			if finished {
				log.Donef("=> TEST FINISHED")
			}
		}
	}
}

func _tryToUploadArchive(uploadURL string, archiveFilePath string) error {
	archFile, err := os.Open(archiveFilePath)
	if err != nil {
		return fmt.Errorf("Failed to open archive file for upload (%s): %s", archiveFilePath, err)
	}
	isFileCloseRequired := true
	defer func() {
		if !isFileCloseRequired {
			return
		}
		if err := archFile.Close(); err != nil {
			log.Printf(" (!) Failed to close archive file (%s): %s", archiveFilePath, err)
		}
	}()

	fileInfo, err := archFile.Stat()
	if err != nil {
		return fmt.Errorf("Failed to get File Stats of the Archive file (%s): %s", archiveFilePath, err)
	}
	fileSize := fileInfo.Size()

	req, err := http.NewRequest("PUT", uploadURL, archFile)
	if err != nil {
		return fmt.Errorf("Failed to create upload request: %s", err)
	}

	req.Header.Add("Content-Length", strconv.FormatInt(fileSize, 10))
	req.ContentLength = fileSize

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("Failed to upload: %s", err)
	}
	isFileCloseRequired = false
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf(" [!] Failed to close response body: %s", err)
		}
	}()

	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("Failed to read response: %s", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("Failed to upload file, response code was: %d", resp.StatusCode)
	}

	return nil
}
