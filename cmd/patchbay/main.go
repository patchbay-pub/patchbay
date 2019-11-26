package main

import (
        "fmt"
        "flag"
        "os"
        "github.com/anderspitman/patchbay"
)

func main() {

        hostCmd := flag.NewFlagSet("host", flag.ExitOnError)
        hostDir := hostCmd.String("dir", ".", "Directory to host")
        rootChannel := hostCmd.String("root-channel", "https://patchbay.pub", "Root channel to host on")
        authToken := hostCmd.String("token", "", "Authentication token")

        if len(os.Args) < 2 {
                fmt.Println("expected command")
                os.Exit(1)
        }

        hostCmd.Parse(os.Args[2:])

        switch os.Args[1] {
        case "host":
                hosterBuilder := patchbay.NewHosterBuilder()
                hoster := hosterBuilder.Dir(*hostDir).RootChannel(*rootChannel).AuthToken(*authToken).Build()
                hoster.Start()
        default:
                fmt.Println("Invalid command:", os.Args[1])
                os.Exit(1)
        }

        ch := make(chan struct{})
        <-ch
}
