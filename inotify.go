package main

import (
    "log"
    "fmt"
    "github.com/howeyc/fsnotify"
)

func main() {
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        log.Fatal(err)
    }

    done := make(chan bool)


    err = watcher.Watch("/home/anil/go/src/muto_work/notify/develop/test")
    if err != nil {
        log.Fatal(err)    
    }

    go func() {
        for {
            select {
            case ev := <-watcher.Event:
                fmt.Println("file(check.txt) has changed changed")
                log.Println("event:", ev)

            case err := <-watcher.Error:
                fmt.Println("file(check.txt) checking has error")
                log.Println("error:", err)
            }
        }
    }()


    <-done
    watcher.Close()
}
