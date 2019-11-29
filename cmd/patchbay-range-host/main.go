package main

import (
        "fmt"
        "log"
        "net/http"
        "os"
        "io"
        "io/ioutil"
        "strings"
        "strconv"
        "flag"
        "path"
)

const RequestPrefix = "Patchbay-Request-"
const ResponsePrefix = "Patchbay-Response-"

type HttpRange struct {
        Start int64 `json:"start"`
        // Note: if end is 0 it won't be included in the json because of omitempty
        End int64 `json:"end,omitempty"`
}

func main() {

        rootChannel := "http://localhost:9001"

        filePath := flag.String("path", "", "File to host")
        flag.Parse()

        client := &http.Client{}

        doneChan := make(chan struct{})

        numWorkers := 4
        for i := 0; i < numWorkers; i++ {
                go func(index int) {
                        for {
                                serveRangeFile(client, rootChannel, filePath)
                                log.Println("Served from worker %d", index)
                        }
                }(i)
        }

        <-doneChan
}

func serveRangeFile(client *http.Client, rootChannel string, filePath *string) {

        filename := path.Base(*filePath)
        url := rootChannel + "/" + filename + "?server=true&doubleclutch=true"
        fmt.Println(url)
        resp, err := client.Post(url, "", nil)
        if err != nil {
                log.Fatal(err)
        }
        defer resp.Body.Close()

        body, err := ioutil.ReadAll(resp.Body)
        if err != nil {
                log.Fatal(err)
        }
        fmt.Println(string(body))

        //patchbayClientHeaders := make(map[string]string)
        patchbayClientHeaders := &http.Header{}

        for k, vList := range resp.Header {
                if strings.HasPrefix(k, RequestPrefix) {
                        // strip the prefix
                        headerName := k[len(RequestPrefix):]
                        for _, v := range vList {
                                patchbayClientHeaders.Add(headerName, v)
                        }
                }
        }

        doubleclutchChannel := resp.Header.Get("Patchbay-Doubleclutch-Channel")

        reqStr := rootChannel + doubleclutchChannel + "?server=true"
        fmt.Println(reqStr)

        file, err := os.Open(*filePath)
        defer file.Close()

        var r *HttpRange
        var req *http.Request

        fileInfo, err := file.Stat()
        if err != nil {
                log.Fatal(err)
        }

        rangeHeader := patchbayClientHeaders.Get("Range")
        if rangeHeader != "" {


                r = parseRange(rangeHeader, fileInfo.Size())

                fmt.Println(r)

                reader := io.NewSectionReader(file, r.Start, r.End - r.Start)

                req, err = http.NewRequest("POST", reqStr, reader)
                if err != nil {
                        log.Fatal(err)
                }

                req.Header.Add(ResponsePrefix + "Content-Range", buildRangeHeader(r, fileInfo.Size()))
                req.Header.Add(ResponsePrefix + "Content-Length", fmt.Sprintf("%d", r.End - r.Start))
                req.Header.Add("Patchbay-Status", "206")
        } else {
                req, err = http.NewRequest("POST", reqStr, file)
                if err != nil {
                        log.Fatal(err)
                }

                //req.Header.Add(ResponsePrefix + "Content-Range", fmt.Sprintf("bytes 0-%d/%d", fileInfo.Size() - 1, fileInfo.Size()))
                req.Header.Add(ResponsePrefix + "Content-Length", fmt.Sprintf("%d", fileInfo.Size()))
        }

        req.Header.Add(ResponsePrefix + "Accept-Ranges", "bytes")
        req.Header.Add(ResponsePrefix + "Content-Type", "video/mp4")
        //req.Header.Add(ResponsePrefix + "Content-Type", "application/octet-stream; charset=utf-8")

        resp, err = client.Do(req)
        if err != nil {
                log.Fatal(err)
        }

        data, err := ioutil.ReadAll(resp.Body)
        if err != nil {
                log.Fatal(err)
        }

        fmt.Println(string(data))
}

func parseRange(header string, size int64) *HttpRange {

        fmt.Println(header)

        // TODO: this is very hacky and brittle
        parts := strings.Split(header, "=")
        rangeParts := strings.Split(parts[1], "-")

        start, err := strconv.Atoi(rangeParts[0])
        if err != nil {
                log.Println("Decode range start failed")
        }

        var end int
        if rangeParts[1] == "" {
                end = int(size)
        } else {
                end, err = strconv.Atoi(rangeParts[1])
                if err != nil {
                        log.Println("Decode range end failed")
                }
        }

        return &HttpRange {
                Start: int64(start),
                End: int64(end),
        }
}

func buildRangeHeader(r *HttpRange, size int64) string {

        if r.End == 0 {
                r.End = size
        }

        contentRange := fmt.Sprintf("bytes %d-%d/%d", r.Start, r.End - 1, size)
        return contentRange
}
