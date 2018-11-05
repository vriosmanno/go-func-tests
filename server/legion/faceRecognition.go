package legion

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	uuid "github.com/Novetta/go.uuid"
	"github.com/Novetta/networking/httputil"
	"github.com/Novetta/security/kerbtypes"
	"github.com/martini-contrib/encoder"
)

var (
	faceRecognitionServerURL = os.Getenv("FACE_RECOGNITION_SERVER_URL")
	faceRecognitionURL       = os.Getenv("FACE_RECOGNITION_URL")
)

// FaceIndexResponse - Contains Hash from being image being indexed / added by Face Index
type FaceIndexResponse struct {
	Hash    string `json:"hash"`
	TheType string `json:"type"`
	TheFile string `json:"file"`
}

// Match - the Metadata on an individual face match
// (Location is top, right, bottom, left format)
type Match struct {
	File     string `json:"file"`
	Location string `json:"location"`
	Type     string `json:"type"`
}

// ErrorReturn - The json encoded error message Face Index returns in the requests
type ErrorReturn struct {
	Title   string `json:"title"`
	Message string `json:"message"`
}

// FacialMatches - Contains Original Image Face Location and an Array of Matches
// (Location is top, right, bottom, left format)
type FacialMatches struct {
	Location []int   `json:"location"`
	Matches  []Match `json:"matches"`
}

// Serves the Face Index/Recognition URL for clients
func getFaceRecognitionURLHandler(r *http.Request, w http.ResponseWriter, enc httputil.JsonEncoder, user *kerbtypes.User) (int, []byte) {
	if len(faceRecognitionURL) > 0 {
		return http.StatusOK, encoder.Must(enc.Encode(faceRecognitionURL))
	}
	return http.StatusBadRequest, encoder.Must(enc.Encode("Face Recognition URL environmental variable is not set"))
}

// IndexFace handles indexing / placing a file into FACE_INDEX
func IndexFace(legionID uuid.UUID, md5Hash string, format string) error {
	var err error

	// Make sure this is of type image
	if strings.ToUpper(format) == "IMAGE" {
		err = postFaceImage(legionID, md5Hash, format)
	}
	return err
}

func postFaceImage(legionID uuid.UUID, md5Hash string, format string) error {
	// Get image file path
	fileImgPath, fileImgName, err := calculateFilePathAndName(md5Hash, fmt.Sprintf("%s%s", md5Hash, GetFileExt(format)))
	if err != nil {
		return fmt.Errorf("Could not determine image file path for %s: %+v", md5Hash, err)
	}

	// Does image file exist?
	TgtFQP := filepath.Join(fileImgPath, fileImgName)
	if _, err := os.Stat(TgtFQP); os.IsNotExist(err) {
		return fmt.Errorf("Specified image file does not exist: %s", TgtFQP)
	}

	// Get image bytes
	fileBytes, err := readFile(TgtFQP)
	if err != nil {
		return fmt.Errorf("Error getting file bytes:  %+v", err)
	}

	// Create multipart form
	buf := &bytes.Buffer{}
	writer := multipart.NewWriter(buf)
	form, err := writer.CreateFormFile("image", fmt.Sprintf("%s%s", md5Hash, GetFileExt(format)))
	if err != nil {
		return fmt.Errorf("Error creating form:  %+v", err)
	}

	// Copy bytes into form
	_, err = io.Copy(form, bytes.NewReader(fileBytes))
	if err != nil {
		return fmt.Errorf("Error copying image to form:  %+v", err)
	}

	// Get content type from form
	contentType := writer.FormDataContentType()

	// Set face index metadata
	// Path: The path to reach the image (scrUrl)
	// Type: The namespace that the imae belongs to
	// Link: The way to route to the entry from gemini (or any other application)
	metaData := map[string]string{
		"path": fmt.Sprintf("files?md5hash=%s&format=%s", md5Hash, format),
		"type": "legion",
		"link": fmt.Sprintf("#/person/%s", legionID.String()),
	}

	// JSON the metadata
	metaDataJSON, err := json.Marshal(metaData)
	if err != nil {
		return fmt.Errorf("Error marshaling metadata to json:  %+v", err)
	}

	writer2, _ := writer.CreateFormField("data")
	_, _ = writer2.Write(metaDataJSON)

	// Close writer
	err = writer.Close()
	if err != nil {
		return fmt.Errorf("Error closing writer:  %+v", err)
	}

	// Parse url and set path
	u, err := url.Parse(faceRecognitionServerURL)
	if err != nil {
		return fmt.Errorf("Error parsing url:  %+v", err)
	}
	u.Path = "/upload"

	// Post to recognition service
	resp, err := http.Post(u.String(), contentType, buf)
	if err != nil {
		return fmt.Errorf("Error sending request to recognition: %+v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		if resp.StatusCode == 404 { // StatusNotFound
			// Check to see if this is face not found
			erfi := ErrorReturn{}
			if err := json.NewDecoder(resp.Body).Decode(&erfi); err != nil {
				return fmt.Errorf("Error decoding response from face index: %+v", err)
			}
			if strings.ToUpper(erfi.Message) == "NO FACES FOUND." {
				// Eventually we want to track that a face was found or not
				return nil
			}
		}

		// An error other than face not found (bad)
		return fmt.Errorf("Error from face index service: %+v", resp)
	}

	// Decode face index service response (This contains a valid md5Hash if a face was found)
	// Eventually we want to track that a face was found or not
	fir := FaceIndexResponse{}
	err = json.NewDecoder(resp.Body).Decode(&fir)
	if err != nil {
		return fmt.Errorf("Error decoding response from face index: %+v", err)
	}

	return nil
}
