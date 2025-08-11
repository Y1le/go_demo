package upload

import (
	"io"
	"os"
	"net/http"
	"net/url"
	"github.com/gin-gonic/gin"
	"log"
	"fmt"
	"mime"
	"path/filepath"
	"github.com/google/uuid"
	"strings" 
)

var invalidChars = []string{"?", "&", "=", "|", "<", ">", ":", "\"", "/", "\\", "*", ".", "$", " "}


func Upload(c *gin.Context) {
    targetURL := c.Query("url")
	requestedName := c.Query("name")
    if targetURL == "" || requestedName == ""{
        c.JSON(http.StatusBadRequest, gin.H{"error": "URL parameter is empty"})
        return
    }

	for _, char := range invalidChars {
		if strings.Contains(requestedName, char) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error" : fmt.Sprintf("Name contains invalid character: '%s'",char),
			})
			return 
		}
	}

    response, err := http.Get(targetURL)
    if err != nil {
        log.Printf("Error fetching URL %s: %v", targetURL, err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch content from URL"})
        return
    }
    defer response.Body.Close() 

    if response.StatusCode != http.StatusOK {
        log.Printf("Received non-OK status code %d from URL %s", response.StatusCode, targetURL)
        c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("Remote server returned status: %d", response.StatusCode)})
        return
    }

    var fileExtension string
    contentType := response.Header.Get("Content-Type")
    if contentType != "" {
        exts, err := mime.ExtensionsByType(contentType)
        if err == nil && len(exts) > 0 {
            fileExtension = exts[0]
        }
    }


    if fileExtension == "" {
        parsedURL, err := url.Parse(targetURL)
        if err == nil {
            pathSegments := strings.Split(parsedURL.Path, "/")
            if len(pathSegments) > 0 {
                potentialFilename := pathSegments[len(pathSegments)-1]
                fileExtension = filepath.Ext(potentialFilename) 
            }
        }
    }

    if fileExtension == "" {
        fileExtension = ".bin"
    }

    uniqueFilename := fmt.Sprintf("%s_%s%s", requestedName, uuid.New().String(), fileExtension)

    uploadDir := "./downloads" // "./uploads"
    if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
        err = os.MkdirAll(uploadDir, 0755)
        if err != nil {
            log.Printf("Error creating directory %s: %v", uploadDir, err)
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create download directory"})
            return
        }
    }
	Path := filepath.Join(uploadDir,strings.TrimPrefix(fileExtension, "."))
    if _, err := os.Stat(Path); os.IsNotExist(err) {
        err = os.MkdirAll(Path, 0755)
        if err != nil {
            log.Printf("Error creating directory %s: %v", Path, err)
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create download directory"})
            return
        }
    }

    filePath := filepath.Join(Path, uniqueFilename)

    outFile, err := os.Create(filePath)//创建文件
    if err != nil {
        log.Printf("Error creating file %s: %v", filePath, err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create file on server"})
        return
    }
    defer outFile.Close()

    bytesCopied, err := io.Copy(outFile, response.Body)
    if err != nil {
        log.Printf("Error saving file %s: %v", filePath, err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
        return
    }
    log.Printf("Downloaded and saved file: %s (%d bytes)", filePath, bytesCopied)

    // 6. 准备将文件内容作为响应返回给客户端
    fileToServe, err := os.Open(filePath)
    if err != nil {
        log.Printf("Error opening saved file %s for serving: %v", filePath, err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to serve file"})
        return
    }
    defer fileToServe.Close()

    extraHeaders := map[string]string{
        "Content-Disposition": fmt.Sprintf(`attachment; filename="%s%s"`, requestedName, fileExtension),
    }

    c.DataFromReader(http.StatusOK, response.ContentLength, response.Header.Get("Content-Type"), fileToServe, extraHeaders)

    os.Remove(filePath)
}