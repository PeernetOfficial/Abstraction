/*
File Name:  Download Transfer.go
Copyright:  2021 Peernet Foundation s.r.o.
Author:     Peter Kleissner

Temporary download code to provide dummy results for testing. To be replaced!
*/

package webapi

import (
    "bytes"
    "os"
    "time"

    "github.com/PeernetOfficial/core/warehouse"
)

// Starts the download.
func (info *DownloadInfo) Start() {
    // current user?
    if bytes.Equal(info.NodeID, info.Backend.SelfNodeID()) {
        info.DownloadSelf()
        return
    }

    for n := 0; n < 3 && info.Peer == nil; n++ {
        _, info.Peer, _ = info.Backend.FindNode(info.NodeID, time.Second*5)

        if info.Status == DownloadCanceled {
            return
        }
    }

    if info.Peer != nil {
        info.Download()
    } else {
        info.Status = DownloadCanceled
    }
}

func (info *DownloadInfo) Download() {
    //fmt.Printf("Download start of %s\n", hex.EncodeToString(info.Hash))

    // try to download the entire File
    reader, fileSize, transferSize, err := FileStartReader(info.Peer, info.Hash, 0, 0, nil)
    if reader != nil {
        defer reader.Close()
    }
    if err != nil {
        info.Status = DownloadCanceled
        return
    } else if fileSize != transferSize {
        info.Status = DownloadCanceled
        return
    }

    info.File.Size = fileSize
    info.Status = DownloadActive

    // download in a loop
    var fileOffset, totalRead uint64
    dataRemaining := fileSize
    readSize := uint64(4096)

    for dataRemaining > 0 {
        //fmt.Printf("data remaining:  downloaded %d from total %d   = %d %%\n", totalRead, fileSize, totalRead*100/fileSize)
        if dataRemaining < readSize {
            readSize = dataRemaining
        }

        data := make([]byte, readSize)
        n, err := reader.Read(data)

        totalRead += uint64(n)
        dataRemaining -= uint64(n)
        data = data[:n]

        if err != nil {
            info.Status = DownloadCanceled
            return
        }

        info.storeDownloadData(data[:n], fileOffset)

        fileOffset += uint64(n)
    }

    //fmt.Printf("data finished:  downloaded %d from total %d   = %d %%\n", totalRead, fileSize, totalRead*100/fileSize)

    info.Finish()
    info.DeleteDefer(time.Hour * 1) // cache the details for 1 hour before removing
}

// Pause pauses the download. Status is DownloadResponseX.
func (info *DownloadInfo) Pause() (status int) {
    info.Lock()
    defer info.Unlock()

    if info.Status != DownloadActive { // The download must be active to be paused.
        return DownloadResponseActionInvalid
    }

    info.Status = DownloadPause

    return DownloadResponseSuccess
}

// Resume resumes the download. Status is DownloadResponseX.
func (info *DownloadInfo) Resume() (status int) {
    info.Lock()
    defer info.Unlock()

    if info.Status != DownloadPause { // The download must be paused to resume.
        return DownloadResponseActionInvalid
    }

    info.Status = DownloadActive

    return DownloadResponseSuccess
}

// Cancel cancels the download. Status is DownloadResponseX.
func (info *DownloadInfo) Cancel() (status int) {
    info.Lock()
    defer info.Unlock()

    if info.Status >= DownloadCanceled { // The download must not be already canceled or finished.
        return DownloadResponseActionInvalid
    }

    info.Status = DownloadCanceled
    info.DiskFile.Handle.Close()

    return DownloadResponseSuccess
}

// Finish marks the download as finished.
func (info *DownloadInfo) Finish() (status int) {
    info.Lock()
    defer info.Unlock()

    if info.Status != DownloadActive { // The download must be active.
        return DownloadResponseActionInvalid
    }

    info.Status = DownloadFinished
    info.DiskFile.Handle.Close()

    return DownloadResponseSuccess
}

// InitDiskFile creates the target File
func (info *DownloadInfo) InitDiskFile(path string) (err error) {
    info.DiskFile.Name = path
    info.DiskFile.Handle, err = os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666) // 666 : All uses can read/write

    return err
}

// storeDownloadData stores downloaded data. It does not change the download Status.
func (info *DownloadInfo) storeDownloadData(data []byte, offset uint64) (status int) {
    info.Lock()
    defer info.Unlock()

    if info.Status != DownloadActive { // The download must be active.
        return DownloadResponseActionInvalid
    }

    if _, err := info.DiskFile.Handle.WriteAt(data, int64(offset)); err != nil {
        return DownloadResponseFileWrite
    }

    info.DiskFile.StoredSize += uint64(len(data))

    return DownloadResponseSuccess
}

func (info *DownloadInfo) DownloadSelf() {
    // Check if the File is available in the local warehouse.
    _, fileSize, status, _ := info.Backend.UserWarehouse.FileExists(info.Hash)
    if status != warehouse.StatusOK {
        info.Status = DownloadCanceled
        return
    }

    info.File.Size = fileSize
    info.Status = DownloadActive

    // read the File
    status, bytesRead, _ := info.Backend.UserWarehouse.ReadFile(info.Hash, 0, int64(info.File.Size), info.DiskFile.Handle)

    info.DiskFile.StoredSize = uint64(bytesRead)

    if status != warehouse.StatusOK {
        info.Status = DownloadCanceled
        return
    }

    info.Finish()
    info.DeleteDefer(time.Hour * 1) // cache the details for 1 hour before removing}
}
