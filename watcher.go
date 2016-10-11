package main

import (
	"flag"
	"fmt"
	"log"
	"log/syslog"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

func init() {
	logwriter, e := syslog.New(syslog.LOG_NOTICE, "Watcher")
	if e == nil {
		log.SetOutput(logwriter)
	}
	log.Print("Starting Watcher!!!")
}

func main() {
	dir := flag.String("dir", "/srv/www", "the directory you want to monitor")
	flag.Parse()

	fmt.Println("Adding Dirs to watch ", *dir)
	err := filepath.Walk(*dir, visit)
	if err != nil {
		log.Fatalln("filepath.Walk() returned %v\n", err)
	}
	fmt.Println("Now Watching")
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				log.Println("event:", event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("modified file:", event.Name)
				}
			case err := <-watcher.Errors:
				log.Println("error:", err)
			}
		}
	}()
	for k := range dirs {
		err := watcher.Add(dirs[k])
		if err != nil {
			log.Fatal(err)
		}
	}
	<-done
}

var dirs []string

func visit(path string, f os.FileInfo, err error) error {
	if f.IsDir() {
		dirs = append(dirs, path)
	}
	return nil
}

//
// func hash_file_md5(filePath string) (string, error) {
// 	var returnMD5String string
// 	file, err := os.Open(filePath)
// 	if err != nil {
// 		return returnMD5String, err
// 	}
// 	defer file.Close()
// 	hash := md5.New()
// 	if _, err := io.Copy(hash, file); err != nil {
// 		return returnMD5String, err
// 	}
// 	hashInBytes := hash.Sum(nil)[:16]
// 	returnMD5String = hex.EncodeToString(hashInBytes)
// 	return returnMD5String, nil
//
// }
