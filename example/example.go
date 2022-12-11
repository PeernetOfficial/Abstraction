package main

import (
    "encoding/json"
    "fmt"
    Abstrations "github.com/PeernetOfficial/Abstraction"
    "github.com/PeernetOfficial/Abstraction/webapi"
    "github.com/PeernetOfficial/core"
    "github.com/google/uuid"
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

    // =================== Searching for the file in the p2p network ====================
    search, err := Abstrations.Search(api, "space")
    if err != nil {
        fmt.Println(err)
    }

    res, _ := PrettyStruct(search)

    // Printing result
    fmt.Println(res)

    // ================== Downloading a file from Peernet ==================

}

func PrettyStruct(data interface{}) (string, error) {
    val, err := json.MarshalIndent(data, "", "    ")
    if err != nil {
        return "", err
    }
    return string(val), nil
}
