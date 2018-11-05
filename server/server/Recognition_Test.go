package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/Novetta/go.logging"
	"github.com/vriosmanno/go-func-tests/server/legion"
)

const (
	//files
	testingDataDirEnv        = "TESTING_DATA"
	fileStoreLocationEnv     = "TESTING_FILE_LOCATION"
	fileStoreTempLocationEnv = "TESTING_FILE_TEMP_LOCATION"
)

var (
	osPermissions  os.FileMode = 0775
	testingDataDir             = ""
	//files
	fileStoreLocation     = ""
	fileStoreTempLocation = ""
)

const version string = "0.1.0"

// go run Recognition_Test.go -m 5974b6a877760f05336d0e9dffb44f61
func main() {
	md5Hash := flag.String("m", "", "jpeg filepath")

	if !flag.Parsed() {
		flag.Parse()
	} else {
		logging.Fatalf("Flags parsed unexpectedly")
	}

	err := checkDirectories()
	if err != nil {
		logging.Fatalf("Error with envirornment variables: %+v", err)
		os.Exit(1)
	}

	if *md5Hash != "" {
		logging.Info("md5Hash: %s", *md5Hash)
		err := legion.UploadImage(*md5Hash)
		if err != nil {
			logging.Errorf("Error uploading image: %+v", err)
		}
	}
}

// Can loops through environment strings to get and create required folders
func checkDirectories() error {
	var err error
	testingDataDir, err = getOrCreateDirByEnvName(testingDataDirEnv)
	if err != nil {
		return err
	}

	fileStoreLocation, err = getOrCreateDirByEnvName(fileStoreLocationEnv)
	if err != nil {
		return err
	}

	fileStoreTempLocation, err = getOrCreateDirByEnvName(fileStoreTempLocationEnv)
	if err != nil {
		return err
	}

	return nil
}

func getOrCreateDirByEnvName(env string) (string, error) {
	dirVal, err := getEnvValue(env)
	if err != nil {
		return dirVal, err
	}

	err = legion.CreateMissingFileDir(dirVal)

	return dirVal, err
}

// Verifies the environment variable exists and returns the string value
func getEnvValue(env string) (string, error) {
	var retVal string

	//validate 'n' value
	if len(env) == 0 {
		return retVal, errors.New("Error: environment variable name can not be empty")
	}

	//pull env val
	retVal, isFound := os.LookupEnv(env)

	if !isFound || len(retVal) == 0 {
		return retVal, errors.New(fmt.Sprintf("Unable to create using environment variable name: %s", env))
	}

	return retVal, nil
}
