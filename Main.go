package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	//"golang.org/x/exp/inotify"
	"time"

	"github.com/howeyc/fsnotify"
)

//*********structure to get json data************

type File struct {
	Dir  string `json:"dir"`
	Exec string `json:"exec"`
	Path string `json:"path"`
}

type config struct {
	Loc         []File `json:"loc"`
	Selfobserve bool   `json:"selfobserve"`
}

//*************function to read json file  (contains direcories tobe monitor and corresponnding action)************
func getFiles(address string) ([]File, bool) {
	raw, err := ioutil.ReadFile(address)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	var c []File
	var config_file config
	json.Unmarshal(raw, &config_file)
	c = config_file.Loc
	return c, config_file.Selfobserve
}

//***********to decode json data*******
func (f File) getStrings() (string, string, string) {
	_, err := json.Marshal(f)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	return f.Dir, f.Exec, f.Path
}

//**********MAIN*************

func main() {
	var err error
	dir, err = filepath.Abs(filepath.Dir(os.Args[0])) //getting file path from run time
	if err != nil {
		log.Fatal(err)
	}
	files, selfobserve := getFiles(dir + "/files.json")

	reqmap = observercreator(files)
	startOberver(selfobserve)
}

var reqmap map[string]observedfile
var dir string

//***********function to get files in maps and call watcher********
func observercreator(files []File) map[string]observedfile {
	//array of structures (to store each file address ,exec function,timer and other functions)
	temp := make(map[string]observedfile)
	for _, f := range files {
		jobj := make(map[string]string)
		m, n, p := f.getStrings()
		if p == "relative" {
			m = dir + m
			n = dir + n
		}
		jobj["dir"] = m
		jobj["exec"] = n
		eventlist := make(map[string]int64) //to initialise Blank eventlist
		newfile := observedfile{eventlist: eventlist, address: jobj}
		temp[m] = newfile

	}
	return temp
}

func startOberver(selfobserve bool) {
	if selfobserve {
		//**************to monitor .json file

		jobj := make(map[string]string)
		eventlist := make(map[string]int64)
		jobj["dir"] = dir + "/files.json"
		jobj["exec"] = dir + "/actions/json.sh"

		jsonfile := observedfile{eventlist: eventlist, address: jobj}

		for _, value := range reqmap {
			go value.watch()
		}
		reqmap[dir+"/files.json"] = jsonfile
		jsonfile.watch()

	} else { //if not to monitor json file
		i := 0
		for _, value := range reqmap {
			if i == len(reqmap) {
				value.watch()
			}
			i++
			go value.watch()
		}
	}
}

//***********structure used for creating attributes to monitor file********
type observedfile struct {
	eventlist map[string]int64  // map to store events("MODIFY","DELETE") and their timestamps
	address   map[string]string //map to store DIR PATH and exec EVENT file path map("dir":"exec")
	mutex     sync.Mutex
	mywatcher *fsnotify.Watcher
}

//************functions accessed by only instance of structure i.e. observedfile
func (obj observedfile) execute() { // executes events of the events File
	len := len(obj.eventlist) //to check either eventlist is empty or not
	if len != 0 {
		if obj.address["dir"] == dir+"/files.json" {
			destroytillnow()
			main()
		}

		fmt.Println("\n\neventlist of ", obj.address["dir"], " :", obj.eventlist, "at", time.Now().Unix())
		out, err := exec.Command(obj.address["exec"]).Output()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("\n\nevent_list :", string(out), " : executed on: ", obj.address["dir"], "at", time.Now().Unix())
		delete(obj.eventlist, "MODIFY")
		delete(obj.eventlist, "DELETE")
	}

}
func destroytillnow() {
	for _, value := range reqmap {
		fmt.Println("REMOVING WATCHER::", value.mywatcher)
		value.mywatcher.RemoveWatch(value.address["dir"])

	}
}

//*************** To synchronize eventlist
func (obj observedfile) sync() {
	<-time.After(time.Millisecond * 200) //wait when an event is fired
	obj.execute()
}

//**********function to implement watcher(fsnotify)************
func (current observedfile) watch() {

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("\n\nwatching: ", current.address["dir"])
	done := make(chan bool)
	err = watcher.Watch(current.address["dir"])
	if err != nil {
		log.Fatal(err)
	}
	current.mywatcher = watcher
	reqmap[current.address["dir"]] = current
	// fmt.Println(current.mywatcher)

	go func() {
		for {
			select {
			case ev := <-watcher.Event:
				//				fmt.Println("\n****************    ", ev.Name, "        ********************")
				if ev.IsModify() {
					//fmt.Println("\nevent:", ev, " at time:", time.Now())
					current.mutex.Lock()
					current.eventlist["MODIFY"] = time.Now().Unix() //update modification timestamp  in structure eventlist
					current.mutex.Unlock()
					current.mutex.Lock()
					current.sync()
					current.mutex.Unlock()
				}
				if ev.IsDelete() {
					//fmt.Println("\nevent:", ev, " at time:", time.Now())
					current.mutex.Lock()
					current.eventlist["DELETE"] = time.Now().Unix() //update deletion timestamp  in structure eventlist
					current.mutex.Unlock()
					watcher.RemoveWatch(current.address["dir"])
					current.mutex.Lock()
					current.sync()
					current.mutex.Unlock()
					go current.watch()
					done <- true
				}
			case err := <-watcher.Error:
				fmt.Println("\nfile checking has error")
				log.Println("\nerror:", err)
			}
		}
	}()
	<-done
	fmt.Println("\nwatcher closed on :", current.address["dir"])
}
