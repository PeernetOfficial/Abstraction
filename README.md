# Abstraction
> **_NOTE:_** This is temporary repo for better abstraction calls for the Peernet core repo. This is expected to be deprecated once the core officially supports strong abstractions.

The following repo was created for the GoAdvent blog post. The objective is to prove how easy it's to extend your Go application with p2p capabilites 
using Peernet. 

### Blog post: https://gopheradvent.com/calendar/2022/peernet-protocol/

## Function calls supported 
### Search a file: 
```go
Abstrations.Search(&<web api object>, "space")
```
### Downloading a file
```go
Abstrations.Download(&<web api object>,<file hash>,<node id>,<download path>)
```
### Add a file to peernet 
```go
Abstrations.Touch(&<web api object>,<file path>)
```

### Remove a file to peernet 
```go
Abstrations.Rm(&<web api object>,<file id>)
```

### An Example workflow can be found here (https://github.com/PeernetOfficial/Abstraction/blob/main/example/example.go)
