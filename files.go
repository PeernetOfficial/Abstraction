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
    "github.com/PeernetOfficial/core"
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
func Touch(b *core.Backend, filePath string) (*TouchReturn, error) {
    // Creates a File in the warehouse
    hash, _, err := b.UserWarehouse.CreateFileFromPath(filePath)
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
    _, publicKey := b.ExportPrivateKey()
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
            } else if _, fileInfo, status, _ := b.UserWarehouse.FileExists(File.Hash); status != warehouse.StatusOK {
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
        if !webapi.SetFileMerkleInfo(b, &blockRecord) {
            return nil, errors.New("merkle information not set")
        }

        filesAdd = append(filesAdd, blockRecord)
    }

    newHeight, newVersion, _ := b.UserBlockchain.AddFiles(filesAdd)

    // Creating object for custom return type
    var touchReturn TouchReturn
    touchReturn.BlockchainHeight = newHeight
    touchReturn.BlockchainVersion = newVersion

    return &touchReturn, nil
}

// Rm Abstracted function that
// removes file from the blockchain and warehouse
func Rm(b *core.Backend, hashStr string) error {
    ID, err := uuid.FromBytes([]byte(hashStr))
    if err != nil {
        return err
    }
    var UUIDs []uuid.UUID
    UUIDs = append(UUIDs, ID)

    _, _, deletedFiles, status := b.UserBlockchain.DeleteFiles(UUIDs)

    // If successfully deleted from the blockchain, delete from the Warehouse in case there are no other references.
    if status == blockchain.StatusOK {
        for n := range deletedFiles {
            if files, status := b.UserBlockchain.FileExists(deletedFiles[n].Hash); status == blockchain.StatusOK && len(files) == 0 {
                b.UserWarehouse.DeleteFile(deletedFiles[n].Hash)
            }
        }
    }

    return nil
}

func
