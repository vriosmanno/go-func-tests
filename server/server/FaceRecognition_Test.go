package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Novetta/go.logging"
	"github.com/vriosmanno/go-func-tests/server/legion"
)

const (
	//files
	fileStoreLocationEnv     = "LEGION_FILE_LOCATION"
	fileStoreTempLocationEnv = "LEGION_FILE_TEMP_LOCATION"
)

var (
	osPermissions os.FileMode = 0775
	legionDataDir             = ""
	//files
	fileStoreLocation     = ""
	fileStoreTempLocation = ""
)

func main() {
	base64String := flag.String("b", "", "base64String")
	fullTest := flag.Bool("f", false, "Complete Full Test")
	imagePath := flag.String("i", "", "jpeg filepath")
	convertBase64 := flag.Bool("c", false, "Convert File Contents To Image")

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

	if *printVersion {
		logging.Info(version)
		os.Exit(0)
	}

	if *base64String != "" {
		Base64ConversionTest(*base64String)
	}

	if *imagePath != "" {
		if *convertBase64 {
			convertedString := legion.ConvertImageBase64(*imagePath)
			logging.Info("Converted Image: %s", convertedString)
		}
		if *fullTest {
			Base64ConversionTest(convertedString)
		}
		if *recognitionServer {

		}
	}
}

func Base64ConversionTest(base64String string) {
	fileName := "test_image"
	tmpFqp, format, err := legion.ConvertBase64toImage(fileName, base64String)

	if err != nil {
		logging.Errorf("Did not receive full qualified path back")
		os.Exit(1)
	}
	logging.Info("Successfully created image at: %s, %s", tmpFqp, format)

	md5hash, err := legion.GetCryptoMd5Hash(tmpFqp)

	if err != nil {
		logging.Errorf("Did not receive md5Hash back")
		os.Exit(1)
	}
	logging.Info("md5Hash returned: %s", md5hash)

	md5Hash, err := legion.IngestFile(tmpFqp, md5hash)

	if err != nil {
		logging.Errorf("Did not receive md5Hash back: %+v", err)
		os.Exit(1)
	}
	logging.Info("md5Hash returned: %s", md5Hash)

	err = legion.IndexFace(md5Hash, filepath.Ext(tmpFqp))
	if err != nil {
		logging.Error(err)
		os.Exit(1)
	}
	logging.Info("Added Image to FaceIndex")
}

// Can loops through environment strings to get and create required folders
func checkDirectories() error {
	var err error
	filesArchiveDir, err = getOrCreateDirByEnvName(filesArchiveEnv)
	if err != nil {
		return err
	}

	filesTempDir, err = getOrCreateDirByEnvName(filesTempEnv)
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
