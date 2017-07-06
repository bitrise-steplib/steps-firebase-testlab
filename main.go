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
	"github.com/bitrise-io/go-utils/sliceutil"
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

// StartTestResponse ...
type StartTestResponse struct {
	Token string `json:"testMatrixId,omitempty"`
}

// TestModel ...
type TestModel struct {
	EnvironmentMatrix EnvironmentMatrixModel `json:"environmentMatrix,omitempty"`
	TestSpecification TestSpecificationModel `json:"testSpecification,omitempty"`
}

// EnvironmentMatrixModel ...
type EnvironmentMatrixModel struct {
	DeviceList *AndroidDevicesList `json:"androidDeviceList,omitempty"`
}

// AndroidDevicesList ...
type AndroidDevicesList struct {
	Devices *[]AndroidDevice `json:"androidDevices,omitempty"`
}

func (deviceList *AndroidDevicesList) addDevice(device AndroidDevice) {
	*deviceList.Devices = append(*deviceList.Devices, device)
}

// AndroidDevice ...
type AndroidDevice struct {
	AndroidModelID   string `json:"androidModelId,omitempty"`
	AndroidVersionID string `json:"androidVersionId,omitempty"`
	Locale           string `json:"locale,omitempty"`
	Orientation      string `json:"orientation,omitempty"`
}

// TestSpecificationModel ...
type TestSpecificationModel struct {
	InstrumentationTest *AndroidInstrumentationTest `json:"androidInstrumentationTest,omitempty"`
	RoboTest            *AndroidRoboTest            `json:"androidRoboTest,omitempty"`
	TestLoop            *AndroidTestLoop            `json:"androidTestLoop,omitempty"`
	TestTimeout         string                      `json:"testTimeout,omitempty"`
}

// AndroidInstrumentationTest ...
type AndroidInstrumentationTest struct {
	AppPackageID    string   `json:"appPackageId,omitempty"`
	TestPackageID   string   `json:"testPackageId,omitempty"`
	TestRunnerClass string   `json:"testRunnerClass,omitempty"`
	TestTargets     []string `json:"testTargets,omitempty"`
}

// RoboDirective ...
type RoboDirective struct {
	ResourceName string `json:"resourceName,omitempty"`
	InputText    string `json:"inputText,omitempty"`
	ActionType   string `json:"actionType,omitempty"` //"ACTION_TYPE_UNSPECIFIED","SINGLE_CLICK","ENTER_TEXT"
}

// AndroidRoboTest ...
type AndroidRoboTest struct {
	AppPackageID       string           `json:"appPackageId,omitempty"`
	AppInitialActivity string           `json:"appInitialActivity,omitempty"`
	MaxDepth           int32            `json:"maxDepth,omitempty"`
	MaxSteps           int32            `json:"maxSteps,omitempty"`
	RoboDirectives     *[]RoboDirective `json:"roboDirectives,omitempty"`
}

// AndroidTestLoop ...
type AndroidTestLoop struct {
	AppPackageID   string   `json:"appPackageId,omitempty"`
	Scenarios      []int32  `json:"scenarios,omitempty"`
	ScenarioLabels []string `json:"scenarioLabels,omitempty"`
}

// TestMatrix ...
type TestMatrix struct {
	State          string `json:"state"` //"TEST_STATE_UNSPECIFIED","VALIDATING","PENDING","RUNNING","FINISHED","ERROR","UNSUPPORTED_ENVIRONMENT","INCOMPATIBLE_ENVIRONMENT","INCOMPATIBLE_ARCHITECTURE","CANCELLED","INVALID"
	TestExecutions []struct {
		TestDetails struct {
			ProgressMessages []string `json:"progressMessages"`
		} `json:"testDetails"`
	}
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

	token := ""

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

		testModel := &TestModel{}
		testModel.EnvironmentMatrix = EnvironmentMatrixModel{DeviceList: &AndroidDevicesList{}}
		testModel.EnvironmentMatrix.DeviceList.Devices = &[]AndroidDevice{}

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
				AndroidModelID:   deviceParams[0],
				AndroidVersionID: deviceParams[1],
				Locale:           deviceParams[2],
				Orientation:      deviceParams[3],
			}

			testModel.EnvironmentMatrix.DeviceList.addDevice(newDevice)
		}

		testModel.TestSpecification = TestSpecificationModel{
			TestTimeout: fmt.Sprintf("%ss", configs.TestTimeout),
		}

		switch configs.TestType {
		case "instrumentation":
			testModel.TestSpecification.InstrumentationTest = &AndroidInstrumentationTest{}
			if configs.AppPackageID != "" {
				testModel.TestSpecification.InstrumentationTest.AppPackageID = configs.AppPackageID
			}
			if configs.InstTestPackageID != "" {
				testModel.TestSpecification.InstrumentationTest.TestPackageID = configs.InstTestPackageID
			}
			if configs.InstTestRunnerClass != "" {
				testModel.TestSpecification.InstrumentationTest.TestRunnerClass = configs.InstTestRunnerClass
			}
			if configs.InstTestTargets != "" {
				targets := strings.Split(strings.TrimSpace(configs.InstTestTargets), ",")
				testModel.TestSpecification.InstrumentationTest.TestTargets = targets
			}
		case "robo":
			testModel.TestSpecification.RoboTest = &AndroidRoboTest{}
			if configs.AppPackageID != "" {
				testModel.TestSpecification.RoboTest.AppPackageID = configs.AppPackageID
			}
			if configs.RoboInitialActivity != "" {
				testModel.TestSpecification.RoboTest.AppInitialActivity = configs.RoboInitialActivity
			}
			if configs.RoboMaxDepth != "" {
				maxDepth, err := strconv.Atoi(configs.RoboMaxDepth)
				if err != nil {
					failf("Failed to parse string(%s) to integer, error: %s", configs.RoboMaxDepth, err)
				}
				testModel.TestSpecification.RoboTest.MaxDepth = int32(maxDepth)
			}
			if configs.RoboMaxSteps != "" {
				maxSteps, err := strconv.Atoi(configs.RoboMaxSteps)
				if err != nil {
					failf("Failed to parse string(%s) to integer, error: %s", configs.RoboMaxSteps, err)
				}
				testModel.TestSpecification.RoboTest.MaxSteps = int32(maxSteps)
			}
			if configs.RoboDirectives != "" {
				roboDirectives := []RoboDirective{}
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
					roboDirectives = append(roboDirectives, RoboDirective{ResourceName: directiveParams[0], InputText: directiveParams[1], ActionType: directiveParams[2]})
				}
				testModel.TestSpecification.RoboTest.RoboDirectives = &roboDirectives
			}
		case "gameloop":
			testModel.TestSpecification.TestLoop = &AndroidTestLoop{}
			if configs.AppPackageID != "" {
				testModel.TestSpecification.TestLoop.AppPackageID = configs.AppPackageID
			}
			if configs.LoopScenarios != "" {
				loopScenarios := []int32{}
				for _, scenarioStr := range strings.Split(strings.TrimSpace(configs.LoopScenarios), ",") {
					scenario, err := strconv.Atoi(scenarioStr)
					if err != nil {
						failf("Failed to parse string(%s) to integer, error: %s", scenarioStr, err)
					}
					loopScenarios = append(loopScenarios, int32(scenario))
				}
				testModel.TestSpecification.TestLoop.Scenarios = loopScenarios
			}
			if configs.LoopScenarioLabels != "" {
				scenarioLabels := strings.Split(strings.TrimSpace(configs.LoopScenarioLabels), ",")
				testModel.TestSpecification.TestLoop.ScenarioLabels = scenarioLabels
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

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			failf("Failed to read response body, error: %s", err)
		}

		responseModel := StartTestResponse{}

		err = json.Unmarshal(body, &responseModel)
		if err != nil {
			failf("Failed to unmarshal response body, error: %s", err)
		}

		token = responseModel.Token

		log.Donef("=> Test started")
	}

	log.Infof("Waiting for test results")
	{
		logsPrinted := []string{}

		finished := false
		for !finished {
			time.Sleep(5 * time.Second)

			url := configs.APIBaseURL + "/" + configs.AppSlug + "/" + configs.BuildSlug + "/" + token

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

			responseModel := &TestMatrix{}

			err = json.Unmarshal(body, responseModel)
			if err != nil {
				failf("Failed to unmarshal response body, error: %s", err)
			}

			finished = (responseModel.State == "FINISHED")

			for i, execution := range responseModel.TestExecutions {
				for _, logContent := range execution.TestDetails.ProgressMessages {
					logSlice := fmt.Sprintf("[TEST %d] => %s", i, logContent)
					if !sliceutil.IsStringInSlice(logSlice, logsPrinted) {
						log.Printf(logSlice)
						logsPrinted = append(logsPrinted, logSlice)
					}
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
