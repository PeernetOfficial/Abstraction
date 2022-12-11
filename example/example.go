package main

import (
    "encoding/json"
    "fmt"
    Abstrations "github.com/PeernetOfficial/Abstraction"
    "github.com/PeernetOfficial/Abstraction/webapi"
    "github.com/PeernetOfficial/core"
    "github.com/google/uuid"
    "os"
    "runtime"
    "time"
)

// Objective is build a sample
// Peernet application using the
// abstracted functions
func main() {
    // Copy config to the appropriate folder
    backend, status, err := core.Init("Test framework/1.0", "Config.yaml", nil, nil)
    if status != core.ExitSuccess {
        fmt.Printf("Error %d initializing backend: %s\n", status, err.Error())
    }

    backend.Connect()

    fmt.Println("Warming up Peernet...")

    api := webapi.Start(backend, []string{""}, false, "", "", 0, 0, uuid.New())

    // 6 seconds
    time.Sleep(time.Millisecond * 6000)

    fmt.Println("========= Peernet Search ===========")

    // =================== Searching for the file in the p2p network ====================
    search, err := Abstrations.Search(api, "space")
    if err != nil {
        fmt.Println(err)
        return
    }

    res, _ := PrettyStruct(search)

    // Printing result
    fmt.Println(res)

    fmt.Println("========= Peernet Download ===========")

    // ================== Downloading a file from Peernet ==================

    // Get download directory
    homeDir, _ := os.UserHomeDir()

    var downloadDir string
    if runtime.GOOS == "darwin" {
        downloadDir = homeDir + "/Downloads/"
    } else {
        downloadDir = homeDir + "\\Downloads\\"
    }

    downloadID, err := Abstrations.Download(api, search.Files[0].Hash, search.Files[0].NodeID, downloadDir+search.Files[0].Name)
    if err != nil {
        fmt.Println(err)
        return
    }

    fmt.Println("Waiting to download file...")

    downloadStatus, err := Abstrations.DownloadStatus(api, downloadID)
    if err != nil {
        fmt.Println(err)
    }

    fmt.Println("========= Downloading file " + downloadStatus.File.Name + " ==============")
    for downloadStatus.DownloadStatus != 5 {
        //fmt.Println(downloadStatus.Progress.Percentage)
        downloadStatus, err = Abstrations.DownloadStatus(api, downloadID)
        if err != nil {
            fmt.Println(err)
        }
    }

    fmt.Println(fmt.Println("========= Download complete =============="))

    fmt.Println("========= Peernet Add file ===========")

    // ================== Add file to peernet ==================
    touch, file, err := Abstrations.Touch(api, "example.go")
    if err != nil {
        fmt.Println(err)
        return
    }

    fmt.Printf("blockchain verison: %v, blockchain height: %v, file hash: %v \n", touch.BlockchainVersion, touch.BlockchainHeight, file.Hash)

    fmt.Println("========= Peernet Remove file ===========")

    // ================== Remove file from peernet ==================
    err = Abstrations.Rm(api, file.ID)
    if err != nil {
        fmt.Println(err)
        return
    }

    fmt.Println("file successfully removed")

}

// PrettyStruct Helper function to print JSON object
func PrettyStruct(data interface{}) (string, error) {
    val, err := json.MarshalIndent(data, "", "    ")
    if err != nil {
        return "", err
    }
    return string(val), nil
}
