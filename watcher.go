package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

func init() {
	log.SetOutput(os.Stdout)
}

var fileStore *FileStateStore

func main() {
	dir := flag.String("dir", "/root/", "the directory you want to monitor")
	clean := flag.Bool("clean", false, "rebuild the database")
	// store := flag.String("store", "store", "the file to store file states")
	flag.Parse()
	fmt.Println("Adding Dirs to watch ", *dir)

	fileStore = NewFileStateStore("./filedb.json")
	if !*clean {
		scandir, err := filepath.Abs(*dir)
		if err != nil {
			log.Fatalln(err)
		}
		err = filepath.Walk(scandir, visit)
		if err != nil {
			log.Printf("filepath.Walk() returned %v\n", err)
		}
	}
	if *clean {
		close(fileStore.save)
		fmt.Println("Rehashing files")
		for key := range fileStore.files {

			hash, err := hash_file_md5(key)
			if err != nil {
				log.Printf("problem hashing file: %s\n", key)
				delete(fileStore.files, key)
			}
			if err == nil {
				info, serr := os.Stat(key)
				if serr != nil {
					log.Printf("problem getting stats on file: %s\n", key)
					delete(fileStore.files, key)
				}
				if serr == nil {
					var fs FileState

					fs.Hash = hash
					fs.LastModified = info.ModTime()
					fs.Path = key
					fileStore.files[key] = fs
				}
			}

			// if err != nil {
			// 	log.Printf("Error adding to clean db %s: %v\n", fs, err)
			// }
		}
		var err = os.Remove("./filedb.json")
		if err != nil {
			log.Fatalln("unable to delete filedb")
		}
		f, err := os.OpenFile("./filedb.json", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		// f, err := os.Open("./filedb.json")
		if err != nil {
			log.Fatalln("unable to write clean db file")
		}

		// b := bufio.NewWriter(f)
		e := json.NewEncoder(f)
		defer f.Close()
		// defer b.Flush()
		for key := range fileStore.files {
			err = e.Encode(fileStore.files[key])
			if err != nil {
				log.Println("FileStateStore Save:", err)
			}
		}

	}

}

func visit(path string, f os.FileInfo, err error) error {

	if !f.IsDir() {
		hash, err := hash_file_md5(path)
		if err != nil {
			log.Printf("Hashing err %v of file %s", err, path)
			return err
		}
		var fs FileState
		fs.Hash = hash
		fs.LastModified = f.ModTime()
		fs.Path = path
		err = fileStore.Put(fs)
		if err != nil {
			return err
		}
	}
	return nil
}

func hash_file_md5(filePath string) (string, error) {
	var returnMD5String string
	file, err := os.Open(filePath)
	if err != nil {
		return returnMD5String, err
	}
	defer file.Close()
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return returnMD5String, err
	}
	hashInBytes := hash.Sum(nil)[:16]
	returnMD5String = hex.EncodeToString(hashInBytes)
	return returnMD5String, nil

}
