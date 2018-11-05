package legion

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"

	logging "github.com/Novetta/go.logging"
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

func UploadImage(md5Hash string) error {
	// Get image file path
	fileImgPath, fileImgName, err := calculateFilePathAndName(md5Hash, md5Hash+".jpg")
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

	// Create multipart form with the mime header "image/jpeg"
	buf := &bytes.Buffer{}
	writer := multipart.NewWriter(buf)
	form, err := CreateImageFormFile(writer, fmt.Sprintf("%s.jpg", md5Hash))

	// Copy bytes into form
	_, err = io.Copy(form, bytes.NewReader(fileBytes))
	if err != nil {
		return fmt.Errorf("Error copying image to form:  %+v", err)
	}

	// Get content type from form
	contentType := writer.FormDataContentType()

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

	if len(rr.Categories) > 0 {
		logging.Info("Recogniton Response Cagtegory: %s ", rr.Categories[0].Category)
	}

	// Update file recognition
	// if len(rr.Categories) > 0 {
	// 	recognition := rr.Categories[0].Category
	// 	err = SetFileRecognition(md5Hash, recognition)
	// 	if err != nil {
	// 		return fmt.Errorf("Error setting file recognition: %+v", err)
	// 	}
	// }

	return nil
}

func CreateImageFormFile(w *multipart.Writer, filename string) (io.Writer, error) {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, "image", filename))
	h.Set("Content-Type", "image/jpeg")
	return w.CreatePart(h)
}
