---
title: "Peernet: An easy way to extend p2p capabilities to your go program"
publishDate: 2022-12-11
authors: ["akilan-selvacoumar"]
---

Today we are going to explore [Peernet](https://peernet.org) a p2p protocol designed for file sharing. But the main objective being simple to use and powerful at the same time. Out of my personal experience extending regular application with p2p capabilities,this has always been a nightmare. The standard problems would be how do I talk to nodes behind NAT, how do I discover nodes in a p2p network etc... If you ask the current community of go developers, IPFS would be considered as the standard solution. When building my 
initial project [p2prc](https://github.com/Akilan1999/p2p-rendering-computation), I did consider trying to extend it with IPFS. The end result was that it was easier writing your own p2p network than extending IPFS with Libp2p (This was a huge nightmare in my case). Encountering Peernet did checkmark most of the reasons 
for a good protocol to extend go applications with p2p capabilities. Peernet is an amazing well abstracted p2p network which utilizes [UDT](https://udt.sourceforge.io) to transfer files with the support of UPNP and UDP hole punching to support talking to nodes behind NAT. There are even more nice features such as a clean decentralized search etc...


## Steps to get stated 
The aim of todays blog post would be to build a sample p2p application using the Peernet Abstraction library. 
The Peernet abstraction library helps extend a go program with p2p capabilities (i.e simple set of function 
calls.)

### Write a the boiler plate code 
This is the go ```main``` function with the Peernet boiler plate code needed to get 
started:
```go
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

// go main function
func main() {
    backend, status, err := core.Init("<application name>/<version no>", "Config.yaml", nil, nil)
    if status != core.ExitSuccess {
        fmt.Printf("Error %d initializing backend: %s\n", status, err.Error())
    }

    backend.Connect()

    api := webapi.Start(backend, []string{"<ip address>:<port no>"}, false, "", "", 0, 0, uuid.New())

    // The abstracted funcitons will be called below here 
}
```

### Set a warm up time 
Before calling the abstracted functions to provide a delay 
so that the go routines in the ```connect``` function can learning about 
nodes in the network. For example below we gave a 6 second 
delay before executing the next function. 
```go
// 6 seconds
time.Sleep(time.Millisecond * 6000)
```

### Searching the p2p network 
It's as simple as calling a function called ```Abstrations.Search(&<web api object>, "<term>")```
to search about files metadata in the p2p network. This function 
returns an search result object. 
```go
search, err := Abstrations.Search(api, "space")
    if err != nil {
        // handle error 
    }
```

### Downloading a file from the p2p network
To download a file it is a simple function call 
called ```Abstrations.Download(&<web api object>,<file hash>,<node id>,<download path>)```. This will ensure the user can download a file from the p2p network based on a hash and node id which is derived from the meta 
data of a file. This meta data can be derived from the search results object as an example. 
```go
// Get download dir of the following machine 
homeDir, _ := os.UserHomeDir()

var downloadDir string
if runtime.GOOS == "darwin" {
    downloadDir = homeDir + "/Downloads/"
} else {
    downloadDir = homeDir + "\\Downloads\\"
}
```
```go
downloadID, err := Abstrations.Download(api, search.Files[0].Hash, search.Files[0].NodeID, downloadDir+search.Files[0].Name)
if err != nil {
    fmt.Println(err)
    return
}
```
In the code snippet above the hash and node id is derived from the first object of the search result array for the search term "space". 

Calling the download function will return a download id, using this ID it's possible to track the status of 
the download. The ```DownloadStatus``` function gets the current status of the current download. 
```go
 downloadStatus, err := Abstrations.DownloadStatus(api, downloadID)
    if err != nil {
        fmt.Println(err)
    }

    fmt.Println("========= Downloading file " + downloadStatus.File.Name + " ==============")
    // The status 5 means that the file has completed downloading 
    for downloadStatus.DownloadStatus != 5 {
        downloadStatus, err = Abstrations.DownloadStatus(api, downloadID)
        if err != nil {
            fmt.Println(err)
        }
    }
```
The code snippet above implements checks if the download is complete and then terminates the loop. 

### Sharing a file from Peernet 
Sharing a file is as simple as calling the function ```touch```. The following function has the 
shape of ```Abstrations.Touch(&<web api object>,<file path>)```. Calling this function will allow 
a user all their file to the [Peernet Warehouse](https://docs.peernet.org/core/warehouse/#limitations) 
and [Peernet blockchain](https://docs.peernet.org/core/blockchain/).
```go
touch, file, err := Abstrations.Touch(api, "example.go")
    if err != nil {
        // handle error
    }
// Printing about information about the file added 
fmt.Printf("blockchain verison: %v, blockchain height: %v, file hash: %v \n", touch.BlockchainVersion, touch.BlockchainHeight, file.Hash)
```

### Removing a file from peernet 
If a file wants to be removed from the [Peernet Warehouse](https://docs.peernet.org/core/warehouse/#limitations) and [Peernet blockchain](https://docs.peernet.org/core/blockchain/) it can be easily 
removed with a single function call. This function consists of the shape ```Abstrations.Rm(&<web api object>,<file id>)```. 
```go
err = Abstrations.Rm(api, file.ID)
    if err != nil {
        // handle error 
    }
```

## More information 
The full source code to abstractions repo can found here
- https://github.com/PeernetOfficial/Abstraction 
> **_NOTE:_** This repo was created particularly for the following blog post. This 
repo will get deprecated once these functions are officially supported in the 
peernet core [repo](https://github.com/PeernetOfficial/core).

The example of the following flow: 
- https://github.com/PeernetOfficial/Abstraction/blob/main/example/example.go

The Peernet protocol source code:
- https://github.com/PeernetOfficial/core

Official Cmd repo:
- https://github.com/PeernetOfficial/Cmd

White paper: 
- https://peernet.org/dl/Peernet%20Whitepaper.pdf
