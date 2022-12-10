/*
File Name:  Download.go
Copyright:  2021 Peernet Foundation s.r.o.
Author:     Peter Kleissner
*/

package webapi

import (
    "encoding/hex"
    "math"
    "net/http"
    "os"
    "strconv"
    "sync"
    "time"

    "github.com/PeernetOfficial/core"
    "github.com/google/uuid"
)

type ApiResponseDownloadStatus struct {
    APIStatus      int       `json:"apistatus"`      // Status of the API call. See DownloadResponseX.
    ID             uuid.UUID `json:"ID"`             // Download ID. This can be used to query the latest Status and take actions.
    DownloadStatus int       `json:"downloadstatus"` // Status of the download. See DownloadX.
    File           ApiFile   `json:"File"`           // File information. Only available for Status >= DownloadWaitSwarm.
    Progress       struct {
        TotalSize      uint64  `json:"totalsize"`      // Total size in bytes.
        DownloadedSize uint64  `json:"downloadedsize"` // Count of bytes download so far.
        Percentage     float64 `json:"percentage"`     // Percentage downloaded. Rounded to 2 decimal points. Between 0.00 and 100.00.
    } `json:"progress"` // Progress of the download. Only valid for Status >= DownloadWaitSwarm.
    Swarm struct {
        CountPeers uint64 `json:"countpeers"` // Count of peers participating in the swarm.
    } `json:"swarm"` // Information about the swarm. Only valid for Status >= DownloadActive.
}

const (
    DownloadResponseSuccess       = 0 // Success
    DownloadResponseIDNotFound    = 1 // Error: Download ID not found.
    DownloadResponseFileInvalid   = 2 // Error: Target File cannot be used. For example, permissions denied to create it.
    DownloadResponseActionInvalid = 4 // Error: Invalid action. Pausing a non-active download, resuming a non-paused download, or canceling already canceled or finished download.
    DownloadResponseFileWrite     = 5 // Error writing File.
)

// Download Status list
const (
    DownloadWaitMetadata = 0 // Wait for File metadata.
    DownloadWaitSwarm    = 1 // Wait to join swarm.
    DownloadActive       = 2 // Active downloading. It could still be stuck at any percentage (including 0%) if no seeders are available.
    DownloadPause        = 3 // Paused by the user.
    DownloadCanceled     = 4 // Canceled by the user before the download finished. Once canceled, a new download has to be started if the File shall be downloaded.
    DownloadFinished     = 5 // Download finished 100%.
)

/*
apiDownloadStart starts the download of a File. The path is the full path on disk to store the File.
The Hash parameter identifies the File to download. The node ID identifies the blockchain (i.e., the "owner" of the File).

Request:    GET /download/start?path=[target path on disk]&Hash=[File Hash to download]&node=[node ID]
Result:     200 with JSON structure ApiResponseDownloadStatus
*/
func (api *WebapiInstance) apiDownloadStart(w http.ResponseWriter, r *http.Request) {
    r.ParseForm()

    // validate hashes, must be blake3
    hash, valid1 := DecodeBlake3Hash(r.Form.Get("Hash"))
    nodeID, valid2 := DecodeBlake3Hash(r.Form.Get("node"))
    if !valid1 || !valid2 {
        http.Error(w, "", http.StatusBadRequest)
        return
    }

    filePath := r.Form.Get("path")
    if filePath == "" {
        http.Error(w, "", http.StatusBadRequest)
        return
    }

    info := &DownloadInfo{Backend: api.Backend, Api: api, ID: uuid.New(), Created: time.Now(), Hash: hash, NodeID: nodeID}

    // create the File immediately
    if info.InitDiskFile(filePath) != nil {
        EncodeJSON(api.Backend, w, r, ApiResponseDownloadStatus{APIStatus: DownloadResponseFileInvalid})
        return
    }

    // add the download to the list
    api.DownloadAdd(info)

    // start the download!
    go info.Start()

    EncodeJSON(api.Backend, w, r, ApiResponseDownloadStatus{APIStatus: DownloadResponseSuccess, ID: info.ID, DownloadStatus: DownloadWaitMetadata})
}

/*
apiDownloadStatus returns the Status of an active download.

Request:    GET /download/Status?ID=[download ID]
Result:     200 with JSON structure ApiResponseDownloadStatus
*/
func (api *WebapiInstance) apiDownloadStatus(w http.ResponseWriter, r *http.Request) {
    r.ParseForm()
    id, err := uuid.Parse(r.Form.Get("ID"))
    if err != nil {
        http.Error(w, "", http.StatusBadRequest)
        return
    }

    info := api.DownloadLookup(id)
    if info == nil {
        EncodeJSON(api.Backend, w, r, ApiResponseDownloadStatus{APIStatus: DownloadResponseIDNotFound})
        return
    }

    info.RLock()

    response := ApiResponseDownloadStatus{APIStatus: DownloadResponseSuccess, ID: info.ID, DownloadStatus: info.Status}

    if info.Status >= DownloadWaitSwarm {
        response.File = info.File

        response.Progress.TotalSize = info.File.Size
        response.Progress.DownloadedSize = info.DiskFile.StoredSize

        response.Progress.Percentage = math.Round(float64(info.DiskFile.StoredSize)/float64(info.File.Size)*100*100) / 100
    }

    if info.Status >= DownloadActive {
        response.Swarm.CountPeers = info.Swarm.CountPeers
    }

    info.RUnlock()

    EncodeJSON(api.Backend, w, r, response)
}

/*
apiDownloadAction pauses, resumes, and cancels a download. Once canceled, a new download has to be started if the File shall be downloaded.
Only active downloads can be paused. While a download is in discovery phase (querying metadata, joining swarm), it can only be canceled.
Action: 0 = Pause, 1 = Resume, 2 = Cancel.

Request:    GET /download/action?ID=[download ID]&action=[action]
Result:     200 with JSON structure ApiResponseDownloadStatus (using APIStatus and DownloadStatus)
*/
func (api *WebapiInstance) apiDownloadAction(w http.ResponseWriter, r *http.Request) {
    r.ParseForm()
    id, err := uuid.Parse(r.Form.Get("ID"))
    action, err2 := strconv.Atoi(r.Form.Get("action"))
    if err != nil || err2 != nil || action < 0 || action > 2 {
        http.Error(w, "", http.StatusBadRequest)
        return
    }

    info := api.DownloadLookup(id)
    if info == nil {
        EncodeJSON(api.Backend, w, r, ApiResponseDownloadStatus{APIStatus: DownloadResponseIDNotFound})
        return
    }

    apiStatus := 0

    switch action {
    case 0: // Pause
        apiStatus = info.Pause()

    case 1: // Resume
        apiStatus = info.Resume()

    case 2: // Cancel
        apiStatus = info.Cancel()
    }

    EncodeJSON(api.Backend, w, r, ApiResponseDownloadStatus{APIStatus: apiStatus, ID: info.ID, DownloadStatus: info.Status})
}

// ---- download tracking ----

type DownloadInfo struct {
    ID           uuid.UUID // Download ID
    Status       int       // Current Status. See DownloadX.
    sync.RWMutex           // Mutext for changing the Status

    // input
    Hash   []byte // File Hash
    NodeID []byte // Node ID of the owner

    // runtime data
    Created time.Time // When the download was Created.
    Ended   time.Time // When the download was finished (only Status = DownloadFinished).

    File ApiFile // File metadata (only Status >= DownloadWaitSwarm)

    DiskFile struct { // Target File on disk to store downloaded data
        Name       string   // File name
        Handle     *os.File // Target File (on disk) to store downloaded data
        StoredSize uint64   // Count of bytes downloaded and stored in the File
    }

    Swarm struct { // Information about the swarm. Only valid for Status >= DownloadActive.
        CountPeers uint64 // Count of peers participating in the swarm.
    }

    // live connections, to be changed
    Peer *core.PeerInfo

    Api     *WebapiInstance
    Backend *core.Backend
}

func (api *WebapiInstance) DownloadAdd(info *DownloadInfo) {
    api.downloadsMutex.Lock()
    api.downloads[info.ID] = info
    api.downloadsMutex.Unlock()
}

func (api *WebapiInstance) DownloadDelete(id uuid.UUID) {
    api.downloadsMutex.Lock()
    delete(api.downloads, id)
    api.downloadsMutex.Unlock()
}

func (api *WebapiInstance) DownloadLookup(id uuid.UUID) (info *DownloadInfo) {
    api.downloadsMutex.Lock()
    info = api.downloads[id]
    api.downloadsMutex.Unlock()
    return info
}

// DeleteDefer deletes the download from the downloads list after the given duration.
// It does not wait for the download to be finished.
func (info *DownloadInfo) DeleteDefer(Duration time.Duration) {
    go func() {
        <-time.After(Duration)
        info.Api.DownloadDelete(info.ID)
    }()
}

// DecodeBlake3Hash decodes a blake3 Hash that is hex encoded
func DecodeBlake3Hash(text string) (hash []byte, valid bool) {
    hash, err := hex.DecodeString(text)
    return hash, err == nil && len(hash) == 256/8
}
