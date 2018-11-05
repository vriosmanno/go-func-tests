package legion

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"net/textproto"

	"github.com/Novetta/go.logging"
)

var (
	recognitionURL = os.Getenv("RECOGNITION_URL")
)

// RecognitionCategory structure to match recognition service
type RecognitionCategory struct {
	Category    string  `json:"category"`
	Description string  `json:"description"`
	Score       float64 `json:"score,omitempty"`
}

// RecognitionResponse structure to match recognition service
type RecognitionResponse struct {
	Categories  []RecognitionCategory `json:"categories"`
	ContentType string                `json:"content-type"`
	ID          string                `json:"id"`
	MD5         string                `json:"md5"`
	Time        string                `json:"time"`
}

// Confirm structure to match recognition service
type Confirm struct {
	Category string `json:"category"`
	MD5      string `json:"md5"`
}

type Category struct {
	Category    CategoryName
	Description string `json:"description"`
}

// CategoryName structure to match recognition service
type CategoryName struct {
	Category string `json:"category"`
}

// CategoryDescription structure to match recognition service
type CategoryDescription struct {
	Category    string `json:"category"`
	Description string `json:"description"`
}

// SuccessResponse structure to match recognition service
type SuccessResponse struct {
	Category    string `json:"category"`
	ContentType string `json:"content-type"`
	ID          string `json:"id"`
	MD5         string `json:"md5"`
	Time        string `json:"time"`
}

// Requires the md5Hash that will find the image
func UploadImage(md5Hash string) error {
	logging.Info("md5Hash: %s", md5Hash)
	// Get image file path
	fPath, fName, err := CalcFilePathAndCheckExists(md5Hash, fmt.Sprintf("%s.jpg", md5Hash))

	// Create multipart form
	buf := &bytes.Buffer{}
	writer := multipart.NewWriter(buf)
	form, err := writer.CreateFormFile("image", fName)
	if err != nil {
		return fmt.Errorf("Error creating form:  %+v", err)
	}

	partHeader := textproto.MIMEHeader{}
	disp := fmt.Sprint("form-data; name=%s; filename=%s", fName)
	partHeader.Add("Content-Disposition", disp)
	partHeader.Add("Content-Type","image/jpeg")
	part, err := writer.CreatePart(partHeader)

	logging.Info("Calculated Filepath: %s%s", fPath, fName)
	// Get image bytes
	fBytes, err := readFile(filepath.Join(fPath, fName))
	if err != nil {
		return fmt.Errorf("Error getting file bytes:  %+v", err)
	}

	// Copy bytes into form
	_, err = io.Copy(form, bytes.NewReader(fBytes))
	if err != nil {
		return fmt.Errorf("Error copying image to form:  %+v", err)
	}

	// Get content type from form
	// contentType := writer.FormDataContentType()
	boundary := writer.Boundary()
	contentType := fmt.Sprintf("image/jpeg; boundary=%s", boundary)
	logging.Info("Content Type: %s", contentType)

	// Close writer
	err = writer.Close()
	if err != nil {
		return fmt.Errorf("Error closing writer:  %+v", err)
	}

	// Parse url and set path
	u, err := url.Parse(recognitionURL)
	if err != nil {
		return fmt.Errorf("Error parsing url:  %+v", err)
	}
	u.Path = "/upload"
	logging.Info("Recognition URL: %s", u)

	// Post to recognition service
	resp, err := http.Post(u.String(), contentType, buf)
	if err != nil {
		return fmt.Errorf("Error sending request to recognition: %+v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("Error from regignition service: %+v, %+v", resp.StatusCode, string(respBody))
	}

	// Decode recognition service response
	rr := RecognitionResponse{}
	err = json.NewDecoder(resp.Body).Decode(&rr)
	if err != nil {
		return fmt.Errorf("Error decoding response from recognition: %+v", err)
	}

	// Update file recognition
	if len(rr.Categories) > 0 {
		// recognition := rr.Categories[0].Category
		// err = SetFileRecognition(md5Hash, recognition)
		for k, v := range rr.Categories {
			fmt.Printf("%s -> %s\n", k, v)
		}
		// if err != nil {
		// 	return fmt.Errorf("Error setting file recognition: %+v", err)
		// }
	}

	return nil
}