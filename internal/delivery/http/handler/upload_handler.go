package handler

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/luqmanulhakimdev/tykket-go/pkg/response"
)

type UploadHandler struct {
	uploadDir string
}

func NewUploadHandler(uploadDir string) *UploadHandler {
	return &UploadHandler{uploadDir: uploadDir}
}

type UploadResult struct {
	Location string `json:"location"`
	URL      string `json:"url"`
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	MimeType string `json:"mimetype"`
}

func (h *UploadHandler) UploadFile(c echo.Context) error {
	file, err := c.FormFile("file")
	if err != nil {
		file, err = c.FormFile("image")
		if err != nil {
			file, err = c.FormFile("upload")
			if err != nil {
				return response.BadRequest(c, "File is required (key: file, image, or upload)", nil)
			}
		}
	}

	// Max 25MB file size limit
	if file.Size > 25*1024*1024 {
		return response.BadRequest(c, "File size exceeds 25MB limit", nil)
	}

	src, err := file.Open()
	if err != nil {
		return response.InternalServerError(c, "Failed to read uploaded file")
	}
	defer src.Close()

	if err := os.MkdirAll(h.uploadDir, 0755); err != nil {
		return response.InternalServerError(c, "Failed to create uploads directory")
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	uniqueID := uuid.New().String()
	sanitizedName := fmt.Sprintf("%d-%s%s", time.Now().Unix(), uniqueID[:8], ext)
	targetPath := filepath.Join(h.uploadDir, sanitizedName)

	dst, err := os.Create(targetPath)
	if err != nil {
		return response.InternalServerError(c, "Failed to save file on disk")
	}
	defer dst.Close()

	if _, err = io.Copy(dst, src); err != nil {
		return response.InternalServerError(c, "Failed to write file contents")
	}

	fileURL := fmt.Sprintf("/uploads/%s", sanitizedName)
	fullURL := fileURL
	scheme := c.Scheme()
	host := c.Request().Host
	if scheme != "" && host != "" {
		fullURL = fmt.Sprintf("%s://%s%s", scheme, host, fileURL)
	}

	result := UploadResult{
		Location: fullURL,
		URL:      fullURL,
		Filename: file.Filename,
		Size:     file.Size,
		MimeType: file.Header.Get("Content-Type"),
	}

	return c.JSON(http.StatusOK, map[string]any{
		"success":  true,
		"message":  "File uploaded successfully",
		"location": fullURL,
		"url":      fullURL,
		"data":     result,
	})
}
