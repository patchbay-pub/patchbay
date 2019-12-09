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
        "math/rand"
        "time"
)

const RequestPrefix = "Pb-Req-"
const ResponsePrefix = "Pb-Res-"

type HttpRange struct {
        Start int64 `json:"start"`
        // Note: if end is 0 it won't be included in the json because of omitempty
        End int64 `json:"end,omitempty"`
}

func main() {

        rand.Seed(time.Now().Unix())

        filePath := flag.String("path", "", "File to host")
        serverFlag := flag.String("server", "https://patchbay.pub", "patchbay server")
        rootChannelFlag := flag.String("root", "/", "Root patchbay channel")
        flag.Parse()

        server := *serverFlag
        rootChannel := *rootChannelFlag

        client := &http.Client{}

        doneChan := make(chan struct{})

        // This isn't the max number of connections, because it forks a
        // goroutine below. This is the max number of long pollers waiting for
        // new connections. I would guess 2-4 should be sufficient for pretty
        // heavy traffic.
        numWorkers := 2
        for i := 0; i < numWorkers; i++ {
                go func(index int) {
                        for {
                                log.Println("Serve from worker", index)
                                serveRangeFile(client, server, rootChannel, filePath, index)
                                log.Println("Fin from worker", index)
                        }
                }(i)
        }

        <-doneChan
}

func serveRangeFile(client *http.Client, server string, rootChannel string, filePath *string, workerId int) {

        //state := ""
        //done := false

        //go func() {
        //        for {
        //                fmt.Println(workerId, state)
        //                if done {
        //                        break
        //                }
        //                time.Sleep(time.Second * 1)
        //        }
        //}()

        filename := path.Base(*filePath)
        url := server + rootChannel + "/" + filename + "?responder=true&switch=true"
        fmt.Println(url, workerId)
        randomChannelId := genRandomChannelId()
        randReader := strings.NewReader(randomChannelId)

        //state = "waiting " + url
        resp, err := client.Post(url, "", randReader)
        if err != nil {
                log.Fatal(err)
        }
        defer resp.Body.Close()

        _, err = ioutil.ReadAll(resp.Body)
        if err != nil {
                log.Fatal(err)
        }
        //fmt.Println(string(body))

        patchbayRequesterHeaders := &http.Header{}

        for k, vList := range resp.Header {
                if strings.HasPrefix(k, RequestPrefix) {
                        // strip the prefix
                        headerName := k[len(RequestPrefix):]
                        for _, v := range vList {
                                patchbayRequesterHeaders.Add(headerName, v)
                        }
                }
        }

        reqStr := server + "/" + randomChannelId + "?responder=true"
        fmt.Println(reqStr, workerId)

        file, err := os.Open(*filePath)
        //defer file.Close()

        var r *HttpRange
        var req *http.Request

        fileInfo, err := file.Stat()
        if err != nil {
                log.Fatal(err)
        }

        rangeHeader := patchbayRequesterHeaders.Get("Range")
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
                req.Header.Add("Pb-Status", "206")
        } else {
                req, err = http.NewRequest("POST", reqStr, file)
                if err != nil {
                        log.Fatal(err)
                }

                //req.Header.Add(ResponsePrefix + "Content-Range", fmt.Sprintf("bytes 0-%d/%d", fileInfo.Size() - 1, fileInfo.Size()))
                req.Header.Add(ResponsePrefix + "Content-Length", fmt.Sprintf("%d", fileInfo.Size()))
        }

        req.Header.Add(ResponsePrefix + "Accept-Ranges", "bytes")
        //req.Header.Add(ResponsePrefix + "Content-Type", "application/octet-stream; charset=utf-8")

        // TODO: this really might not be safe, but need to move forward.
        // Think more deeply about this at some point. It was necessary
        // because when streaming videos, Firefox and Chrome were both
        // getting in a state where the request was blocked here. I suspect
        // it was happening because they were opening a connection but never
        // reading the response (ie while the video was paused). But this
        // prevented other requesters from accessing the resource. Doing it in
        // a goroutine like this essentially forks it, so it doesn't matter if
        // some of them are sitting around doing nothing.
        go func() {
                //state = "waiting for data " + reqStr
                resp, err = client.Do(req)
                if err != nil {
                        log.Fatal(err)
                }

                data, err := ioutil.ReadAll(resp.Body)
                if err != nil {
                        log.Fatal(err)
                }

                fmt.Println(string(data))

                //done = true

                file.Close()
        }()
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

const channelChars string = "0123456789abcdefghijkmnpqrstuvwxyz";
func genRandomChannelId() string {
        channelId := ""
        for i := 0; i < 32; i++ {
                channelId += string(channelChars[rand.Intn(len(channelChars))])
        }
        return channelId
}
