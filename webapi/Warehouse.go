/*
File Name:  Warehouse.go
Copyright:  2021 Peernet Foundation s.r.o.
Author:     Peter Kleissner
*/

package webapi

import (
    "net/http"
    "strconv"

    "github.com/PeernetOfficial/core/warehouse"
)

// WarehouseResult is the response to creating a new File in the warehouse
type WarehouseResult struct {
    Status int    `json:"Status"` // See warehouse.StatusX.
    Hash   []byte `json:"Hash"`   // Hash of the File.
}

/*
apiWarehouseCreateFile creates a File in the warehouse.

Request:    POST /warehouse/create with raw data to create as new File
Response:   200 with JSON structure WarehouseResult
*/
func (api *WebapiInstance) apiWarehouseCreateFile(w http.ResponseWriter, r *http.Request) {
    hash, status, err := api.Backend.UserWarehouse.CreateFile(r.Body, 0)

    if err != nil {
        api.Backend.LogError("warehouse.CreateFile", "Status %d error: %v", status, err)
    }

    EncodeJSON(api.Backend, w, r, WarehouseResult{Status: status, Hash: hash})
}

/*
apiWarehouseCreateFilePath creates a File in the warehouse by copying it from an existing File.
Warning: An attacker could supply any local File using this function, put them into storage and read them! No input path verification or limitation is done.
In the future the API should be secured using a random API key and setting the CORS header prohibiting regular browsers to access the API.

Request:    GET /warehouse/create/path?path=[target path on disk]
Response:   200 with JSON structure WarehouseResult
*/
func (api *WebapiInstance) apiWarehouseCreateFilePath(w http.ResponseWriter, r *http.Request) {
    r.ParseForm()
    filePath := r.Form.Get("path")
    if filePath == "" {
        http.Error(w, "", http.StatusBadRequest)
        return
    }

    hash, status, err := api.Backend.UserWarehouse.CreateFileFromPath(filePath)

    if err != nil {
        api.Backend.LogError("warehouse.CreateFile", "Status %d error: %v", status, err)
    }

    EncodeJSON(api.Backend, w, r, WarehouseResult{Status: status, Hash: hash})
}

/*
apiWarehouseReadFile reads a File in the warehouse.

Request:    GET /warehouse/read?Hash=[Hash]
            Optional parameters &offset=[File offset]&limit=[read limit in bytes]
Response:   200 with the raw File data
			404 if File was not found
			500 in case of internal error opening the File
*/
func (api *WebapiInstance) apiWarehouseReadFile(w http.ResponseWriter, r *http.Request) {
    r.ParseForm()
    hash, valid1 := DecodeBlake3Hash(r.Form.Get("Hash"))
    if !valid1 {
        http.Error(w, "", http.StatusBadRequest)
        return
    }

    offset, _ := strconv.Atoi(r.Form.Get("offset"))
    limit, _ := strconv.Atoi(r.Form.Get("limit"))

    status, bytesRead, err := api.Backend.UserWarehouse.ReadFile(hash, int64(offset), int64(limit), w)

    switch status {
    case warehouse.StatusFileNotFound:
        w.WriteHeader(http.StatusNotFound)
        return
    case warehouse.StatusInvalidHash, warehouse.StatusErrorOpenFile, warehouse.StatusErrorSeekFile:
        w.WriteHeader(http.StatusInternalServerError)
        return
        // Cannot catch warehouse.StatusErrorReadFile since data may have been already returned.
        // In the future a special header indicating the expected File length could be sent (would require a callback in ReadFile), although the caller should already know the File size based on metadata.
    }

    if err != nil {
        api.Backend.LogError("warehouse.ReadFile", "Status %d read %d error: %v", status, bytesRead, err)
    }
}

/*
apiWarehouseDeleteFile deletes a File in the warehouse.

Request:    GET /warehouse/delete?Hash=[Hash]
Response:   200 with JSON structure WarehouseResult
*/
func (api *WebapiInstance) apiWarehouseDeleteFile(w http.ResponseWriter, r *http.Request) {
    r.ParseForm()
    hash, valid1 := DecodeBlake3Hash(r.Form.Get("Hash"))
    if !valid1 {
        http.Error(w, "", http.StatusBadRequest)
        return
    }

    status, err := api.Backend.UserWarehouse.DeleteFile(hash)

    if err != nil {
        api.Backend.LogError("warehouse.DeleteFile", "Status %d error: %v", status, err)
    }

    EncodeJSON(api.Backend, w, r, WarehouseResult{Status: status, Hash: hash})
}

/*
apiWarehouseReadFilePath reads a File from the warehouse and stores it to the target File. It fails with StatusErrorTargetExists if the target File already exists.
The path must include the full directory and File name.

Request:    GET /warehouse/read/path?Hash=[Hash]&path=[target path on disk]
            Optional parameters &offset=[File offset]&limit=[read limit in bytes]
Response:   200 with JSON structure WarehouseResult
*/
func (api *WebapiInstance) apiWarehouseReadFilePath(w http.ResponseWriter, r *http.Request) {
    r.ParseForm()
    hash, valid1 := DecodeBlake3Hash(r.Form.Get("Hash"))
    if !valid1 {
        http.Error(w, "", http.StatusBadRequest)
        return
    }

    targetFile := r.Form.Get("path")
    offset, _ := strconv.Atoi(r.Form.Get("offset"))
    limit, _ := strconv.Atoi(r.Form.Get("limit"))

    status, bytesRead, err := api.Backend.UserWarehouse.ReadFileToDisk(hash, int64(offset), int64(limit), targetFile)

    if err != nil {
        api.Backend.LogError("warehouse.ReadFileToDisk", "Status %d read %d error: %v", status, bytesRead, err)
    }

    EncodeJSON(api.Backend, w, r, WarehouseResult{Status: status, Hash: hash})
}
