package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/cspor/go-practice-files/config"
	"github.com/cspor/go-practice-files/models/row"
	"github.com/cspor/go-practice-files/services/errorHandler"
	"github.com/cspor/go-practice-files/services/filesystem"
	"github.com/google/uuid"
	"net/http"
	"os"
	"sync"
	"time"
)

func main() {
	fmt.Println("Go web app started on port: 3000")

	setupRoutes()

	http.ListenAndServe(":3000", nil)
}

func homePage(writer http.ResponseWriter, request *http.Request) {
	fmt.Fprintf(writer, "My Go Application")
}

func setupRoutes() {
	http.HandleFunc("/", homePage)
	http.HandleFunc("/files", files)
}

func files(writer http.ResponseWriter, request *http.Request) {
	fmt.Fprintf(writer, "Starting to write %d pages\n", config.PageCount)
	fmt.Fprintf(writer, "Starting to write %d rows \n", config.RowCount)

	id, _ := uuid.NewRandom()

	idString := id.String()
	pagesFolder := config.PagesFolder + idString
	buildsFolder := config.BuildsFolder + idString

	var waitGroup = sync.WaitGroup{}

	filesystem.RemakeFolder(pagesFolder)
	filesystem.RemakeFolder(buildsFolder)

	pagesStart := time.Now()

	writePages(pagesFolder, config.PageCount, config.RowCount, &waitGroup, writer)

	waitGroup.Wait()

	took("Creating pages", pagesStart, writer)

	// write all files in source directory to destination
	writeStart := time.Now()
	filesystem.WriteFilesInDirToDestination(pagesFolder, buildsFolder, "export_write")
	took("Writing to export", writeStart, writer)

	// copy all files in source directory to destination
	copyStart := time.Now()
	filesystem.CopyFilesInDirToDestination(pagesFolder, filesystem.OpenFileToAppend(buildsFolder, "export_copy"))
	took("Copying to export", copyStart, writer)

	// Cleanup
	err := os.RemoveAll(pagesFolder)
	errorHandler.Check(err)
	e := os.RemoveAll(buildsFolder)
	errorHandler.Check(e)
	fmt.Fprintf(writer, "Cleaned up")
}

// writePages Writes rowCount rows to pageCount pages
func writePages(pagesFolder string, pageCount int, rowCount int, waitGroup *sync.WaitGroup, writer http.ResponseWriter) {
	for index := 1; index <= pageCount; index++ {
		waitGroup.Add(1)
		go writeUUIDsToFile(pagesFolder, fmt.Sprint("page_", index), rowCount, waitGroup, writer)
	}
}

// writeUUIDsToFile writes count Rows to the file
func writeUUIDsToFile(folderName string, fileName string, count int, waitGroup *sync.WaitGroup, writer http.ResponseWriter) {
	file := filesystem.OpenFileToAppend(folderName, fileName)

	bufferedWriter := bufio.NewWriter(file)

	// Write new rows to the file
	for index := 1; index <= count; index++ {
		rowJson, err := json.Marshal(row.NewRow())
		errorHandler.Check(err)

		bytesCount, err := bufferedWriter.Write(rowJson)
		_ = bytesCount
		errorHandler.Check(err)

		bufferedWriter.WriteString("\n")
	}

	e := bufferedWriter.Flush()
	errorHandler.Check(e)

	fmt.Fprintf(writer, "finished writing to %s\n", fileName)

	errorHandler.Check(file.Close())

	waitGroup.Done()
}

func took(message string, timer time.Time, writer http.ResponseWriter) {
	fmt.Fprintf(writer, message+" took: %s \n", time.Since(timer))
}
