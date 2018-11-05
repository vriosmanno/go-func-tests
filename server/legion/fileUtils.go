package legion

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	datatypes "github.com/Novetta/go.datatypes"
	"github.com/Novetta/go.logging"
	uuid "github.com/Novetta/go.uuid"
	"github.com/Novetta/networking/httputil"
	"github.com/Novetta/security/kerbtypes"

	"github.com/martini-contrib/encoder"
)

var (
	imageFormat               = ".jpg"
	osMode                    = os.O_CREATE | os.O_WRONLY
	osPermissions os.FileMode = 0775
	fileStoreLocation = os.Getenv("TESTING_FILE_LOCATION")
	fileStoreTempLocation = os.Getenv("TESTING_FILE_TEMP_LOCATION")
)

type fileInstanceErrorReturn datatypes.ErrorReturn

func (e fileInstanceErrorReturn) WithMessage(msg string) fileInstanceErrorReturn {
	return fileInstanceErrorReturn{
		Title:   "File Utils Error",
		Message: msg}
}

// Gets the file path and checks if it exists
func CalcFilePathAndCheckExists(md5Hash string, ext string) (string, string, error) {
	logging.Info("Calculated Filename: %s", fmt.Sprintf("%s.jpg", md5Hash))
	filePath, fileName, err := calculateFilePathAndName(md5Hash, fmt.Sprintf("%s.jpg", md5Hash))
	if err != nil {
		return "", "", fmt.Errorf("Could not determine image file path for %s: %+v", md5Hash, err)
	}
	logging.Info("FilePath: %s", filepath.Join(filePath, fileName))
	err = fileExists(filepath.Join(filePath, fileName))
	if err != nil {
		return "", "", err
	}
	return filePath, fileName, nil
}

// Checks if a file exists
func fileExists(TgtFQP string) error {
	if _, err := os.Stat(TgtFQP); os.IsNotExist(err) {
		return fmt.Errorf("Specified image file does not exist: %s", TgtFQP)
	}
	return nil
}

// This calculates the file path based on the md5hash and returns the fully qualified path split into file path and file name
// Example:
//   md5Hash: c9f0a50243285ecdee9cd88f9db86730
//   fileStoreLocation: /data/aide/legion
//   ext: .jpg
//   newFilePath = /data/aide/legion/c9/f0/
//   newFileName = c9f0a50243285ecdee9cd88f9db86730.jpg
func calculateFilePathAndName(md5Hash string, ext string) (string, string, error) {
	if len(md5Hash) < 4 {
		return "", "", fmt.Errorf("Invalid hash length: %d", len(md5Hash))
	}

	newFilePath := filepath.Join(fileStoreLocation, filepath.Join(md5Hash[0:2], md5Hash[2:4]))
	newFileName := md5Hash + filepath.Ext(ext)

	return newFilePath, newFileName, nil
}

// Takes a base64 string and a filename to create a temporary file for ingesting in to legion
func ConvertBase64toImage(legionID uuid.UUID, base64String string) (string, string, error) {
	if base64String == "" {
		return "", "", fmt.Errorf("Missing arguments: legionID: %s - base64String: %+v", legionID.String(), base64String)
	}

	// Convert Base64 to raw bytes
	rawBytes, err := base64.StdEncoding.DecodeString(base64String)
	if err != nil {
		return "", "", fmt.Errorf("Error decoding base64: %+v.", err.Error())
	}

	rawImage, format, _ := image.Decode(bytes.NewReader(rawBytes))

	// This fully qualified path is the temp location for the file using the legionID and file format.
	fqp := filepath.Join(fileStoreTempLocation, fmt.Sprintf("%s%s", legionID.String(), imageFormat))
	file, _ := os.OpenFile(fqp, osMode, osPermissions)

	defer file.Close()

	var opt jpeg.Options
	opt.Quality = 100

	logging.Debug("Creating file: %s", fqp)

	switch format {
	case "png":
		rawImage, _ := png.Decode(bytes.NewReader(rawBytes))
		err = jpeg.Encode(file, ConvertPngToJpeg(rawImage), &opt)
	case "jpeg":
		err = jpeg.Encode(file, rawImage, &opt)
	default:
		return "", "", fmt.Errorf("Encoding not recognized for image, Skipping...")
	}
	if err != nil {
		return "", "", fmt.Errorf("Error encoding image: %+v", err)
	}

	logging.Debug("Created image: %s", fqp)

	return fqp, "IMAGE", nil
}

// Converts image from base64 to an image, ingests the Image to the file system and registers it to face index
func ConvertAndIngestImage(legionID uuid.UUID, base64String string) (string, string, error) {
	tmpFqp, format, err := ConvertBase64toImage(legionID, base64String)
	if err != nil {
		return "", "", fmt.Errorf("Error converting Base64 to Image: %+v", err)
	}

	// Get the md5 hash for the file
	md5hash, err := GetCryptoMd5Hash(tmpFqp)
	if err != nil {
		return "", "", fmt.Errorf("Error getting md5hash: %+v", err)
	}

	// Ingest the file into legion
	err = IngestFile(tmpFqp, md5hash)
	if err != nil {
		return "", "", fmt.Errorf("Error Ingesting file: %+v", err)
	}

	// This can return an error and ingest is still successful
	// Would return error if face was not detected
	err = IndexFace(legionID, md5hash, format)
	if err != nil {
		logging.Error(err)
	}

	return md5hash, format, nil
}

// This is a testing function that also allows us to create the base64 strings from any image.
func ConvertImageToBase64(imagePath string) string {
	imageBytes, err := ioutil.ReadFile(imagePath)
	if err != nil {
		logging.Errorf("Unable to read file: %+v", err)
		os.Exit(1)
	}

	imgBase64Str := base64.StdEncoding.EncodeToString(imageBytes)

	return imgBase64Str
}

// Converts a PNG image to a JPEG in memory
func ConvertPngToJpeg(imgSrc image.Image) image.Image {
	// Create a new image from the decoded image
	newImg := image.NewRGBA(imgSrc.Bounds())

	// Use a white background to replace any transparent background
	draw.Draw(newImg, newImg.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)

	// Paste PNG over the white background
	draw.Draw(newImg, newImg.Bounds(), imgSrc, imgSrc.Bounds().Min, draw.Over)

	return newImg
}

// Checks if the file directory exists and if not then creates it
func CreateMissingFileDir(filepath string) error {
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		err = os.MkdirAll(filepath, osPermissions)
		return err
	}
	return nil
}

// Checks based on format what file extension should be provided
// this could later be used for querying the database to retrieve this
func GetFileExt(format string) string {
	var retVal string
	switch format {
	case "IMAGE":
		retVal = ".jpg"
	default:
		retVal = ""
	}

	return retVal
}

// The endpoint handler for files
func getFileInstanceHandler(w http.ResponseWriter, enc httputil.JsonEncoder, r *http.Request, user *kerbtypes.User) {
	errorMsg := fileInstanceErrorReturn{}.WithMessage("Unable to get file instances.")
	md5Hash := r.URL.Query().Get("md5hash")
	format := r.URL.Query().Get("format")
	if len(md5Hash) == 0 || len(format) == 0 {
		logging.Errorf("The md5hash and format cannot be empty: %s - %s", md5Hash, format)
		w.WriteHeader(http.StatusBadRequest)
		w.Write(encoder.Must(enc.Encode(errorMsg)))
		return
	}

	filePathExt := GetFileExt(format)

	filePath, fileName, err := calculateFilePathAndName(md5Hash, filePathExt)
	if err != nil {
		logging.Errorf("Could not calculate file path: %s - %+v", md5Hash, err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write(encoder.Must(enc.Encode(errorMsg)))
		return
	}

	fqp := filepath.Join(filePath, fileName)
	http.ServeFile(w, r, fqp)
	return
}

// Calculates folder path based on md5hash and moves file to that directory
func IngestFile(oldFqp string, md5Hash string) error {
	newFilePath, newFileName, err := calculateFilePathAndName(md5Hash, filepath.Ext(oldFqp))
	if err != nil {
		return err
	}

	newFqp := filepath.Join(newFilePath, newFileName)
	logging.Debug("New Full Qualified Path: %s", newFqp)

	// If the file does not already exist then ingest otherwise delete the temp file
	if _, err := os.Stat(newFqp); os.IsNotExist(err) {
		f, err := os.Open(oldFqp)
		if err != nil {
			return err
		}
		defer f.Close()

		// Create any missing directories
		err = CreateMissingFileDir(newFilePath)
		if err != nil {
			return fmt.Errorf("Unable to create new directory: %+v", err)
		}

		// Rename and relocate file
		err = os.Rename(oldFqp, newFqp)
		if err != nil {
			return fmt.Errorf("Unable to move file to location: %s - %+v", newFilePath, err)
		}
	} else {
		err = os.Remove(oldFqp)
		if err != nil {
			return fmt.Errorf("Unable to delete file: %+v", err)
		}
	}

	return nil
}

func readFile(filePath string) ([]byte, error) {
	logging.Info("Read file path: %s", filePath)
	bytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return bytes, fmt.Errorf("ReadFile error while reading bytes from path %s: %+v", filePath, err)
	}
	return bytes, nil
}
