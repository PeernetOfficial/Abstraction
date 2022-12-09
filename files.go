/*
File Name:  abstractions.go
Copyright:  2021 Peernet s.r.o.
Authors: Peter Kleissner, Akilan Selvacoumar
*/

package Abstrations

import (
    "encoding/hex"
    "errors"
    "github.com/PeernetOfficial/Abstraction/webapi"
    "github.com/PeernetOfficial/core/blockchain"
    "github.com/PeernetOfficial/core/protocol"
    "github.com/PeernetOfficial/core/warehouse"
    "github.com/google/uuid"
    "path/filepath"
    "time"
)

/*
Library description
to about abstracted function to easily add and remove files.
*/

type TouchReturn struct {
    BlockchainHeight  uint64
    BlockchainVersion uint64
}

// Touch abstracted function that creates a file
// and adds the file to the warehouse and
// blockchain
// returns blockchain version and height
func Touch(api *webapi.WebapiInstance, filePath string) (*TouchReturn, error) {
    // Creates a File in the warehouse
    hash, _, err := api.Backend.UserWarehouse.CreateFileFromPath(filePath)
    if err != nil {
        return nil, err
    }

    // Add the File to the local blockchain
    var input webapi.ApiBlockAddFiles
    var inputFiles []webapi.ApiFile
    var inputFile webapi.ApiFile

    // Write File information to the input File
    inputFile.Date = time.Now()
    // Folder and File name
    dir, file := filepath.Split(filePath)
    inputFile.Folder = dir
    inputFile.Name = file
    inputFile.ID = uuid.New()
    inputFile.Hash = hash

    // Get the public key of the current node
    _, publicKey := api.Backend.ExportPrivateKey()
    inputFile.NodeID = []byte(hex.EncodeToString(publicKey.SerializeCompressed()))

    inputFiles = append(inputFiles, inputFile)

    input.Files = inputFiles

    var filesAdd []blockchain.BlockRecordFile

    for _, File := range input.Files {
        if len(File.Hash) != protocol.HashSize {
            return nil, errors.New("bad request")
        }
        if File.ID == uuid.Nil { // if the ID is not provided by the caller, set it
            File.ID = uuid.New()
        }

        // Verify that the File exists in the warehouse. Folders are exempt from this check as they are only virtual.
        if !File.IsVirtualFolder() {
            if _, err := warehouse.ValidateHash(File.Hash); err != nil {
                return nil, errors.New("bad request when validating hash")
            } else if _, fileInfo, status, _ := api.Backend.UserWarehouse.FileExists(File.Hash); status != warehouse.StatusOK {
                //EncodeJSON(api.backend, w, r, apiBlockchainBlockStatus{Status: blockchain.StatusNotInWarehouse})
                return nil, errors.New("file not in warehouse")
            } else {
                File.Size = fileInfo
            }
        } else {
            File.Hash = protocol.HashData(nil)
            File.Size = 0
        }

        blockRecord := webapi.BlockRecordFileFromAPI(File)

        // Set the merkle tree info as appropriate.
        if !webapi.SetFileMerkleInfo(api.Backend, &blockRecord) {
            return nil, errors.New("merkle information not set")
        }

        filesAdd = append(filesAdd, blockRecord)
    }

    newHeight, newVersion, _ := api.Backend.UserBlockchain.AddFiles(filesAdd)

    // Creating object for custom return type
    var touchReturn TouchReturn
    touchReturn.BlockchainHeight = newHeight
    touchReturn.BlockchainVersion = newVersion

    return &touchReturn, nil
}

// Rm Abstracted function that
// removes file from the blockchain and warehouse
func Rm(api *webapi.WebapiInstance, hashStr string) error {
    ID, err := uuid.FromBytes([]byte(hashStr))
    if err != nil {
        return err
    }
    var UUIDs []uuid.UUID
    UUIDs = append(UUIDs, ID)

    _, _, deletedFiles, status := api.Backend.UserBlockchain.DeleteFiles(UUIDs)

    // If successfully deleted from the blockchain, delete from the Warehouse in case there are no other references.
    if status == blockchain.StatusOK {
        for n := range deletedFiles {
            if files, status := api.Backend.UserBlockchain.FileExists(deletedFiles[n].Hash); status == blockchain.StatusOK && len(files) == 0 {
                api.Backend.UserWarehouse.DeleteFile(deletedFiles[n].Hash)
            }
        }
    }

    return nil
}

// Search Abstracted function
// to query for files available
// in the p2p network (i.e the
// Peernet protocol)
// Since it's default it's ran for
// 5 seconds as the default timeout
// (This will be changed on later
// iterations)
func Search(api *webapi.WebapiInstance, term string) (*webapi.SearchResult, error) {
    var input webapi.SearchRequest
    input.Term = term
    input.Timeout = 5
    jobID, err := StartSearch(api, &input)
    if err != nil {
        return nil, err
    }

    // 6 seconds
    time.Sleep(1000 * 6)

    result, err := SearchResult(api, jobID)
    if err != nil {
        return nil, err
    }

    return result, nil
}

// StartSearch Abstracted function that
// starts the search job based on specified
// parameters and return the job ID
// for a reference
func StartSearch(api *webapi.WebapiInstance, input *webapi.SearchRequest) (uuid.UUID, error) {
    if input.Timeout <= 0 {
        input.Timeout = 20
    }
    if input.MaxResults <= 0 {
        input.MaxResults = 200
    }

    // Terminate previous searches, if their IDs were supplied. This allows terminating the old search immediately without making a separate /search/terminate request.
    for _, terminate := range input.TerminateID {
        if job := api.JobLookup(terminate); job != nil {
            job.Terminate()
            api.RemoveJob(job)
        }
    }

    job := api.DispatchSearch(*input)

    return job.ID, nil
}

func SearchResult(api *webapi.WebapiInstance, jobID uuid.UUID) (*webapi.SearchResult, error) {
    // find the job ID
    job := api.JobLookup(jobID)
    if job == nil {
        return nil, errors.New("job id not found")
    }

    limit := 100

    // query all results
    var resultFiles []*webapi.ApiFile

    resultFiles = job.ReturnNext(limit)

    var result webapi.SearchResult
    result.Files = []webapi.ApiFile{}

    // loop over results
    for n := range resultFiles {
        result.Files = append(result.Files, *resultFiles[n])
    }

    // set the status
    if len(result.Files) > 0 {
        if job.IsSearchResults() {
            result.Status = 0 // 0 = Success with results
            return &result, nil
        } else {
            result.Status = 1 // No more results to expect
            return nil, errors.New("no more results to expect (Search still running)")
        }
    } else {
        switch job.Status {
        case webapi.SearchStatusLive:
            result.Status = 3 // No results yet available keep trying
            return nil, errors.New("no results yet available keep trying")
        case webapi.SearchStatusTerminated:
            result.Status = 1 // No more results to expect
            return nil, errors.New("no more results to expect (Search terminated)")
        default: // SearchStatusNoIndex, SearchStatusNotStarted
            result.Status = 1 // No more results to expect
            return nil, errors.New("no more results to expect (Search not started)")
        }
    }

    return nil, errors.New("search not successful")
}
