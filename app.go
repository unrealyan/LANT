package main

import (
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

type file struct {
	Name string `json:"name"`
	File os.File
}

func openBrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		cmd := exec.Command("open", url)
		err = cmd.Start()
		err = cmd.Wait()
		fmt.Println(err)

	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Fatal(err)
	}
}

func upload(writer http.ResponseWriter, r *http.Request) {

	r.ParseMultipartForm(200)
	mForm := r.MultipartForm

	for k, _ := range mForm.File {

		file, fileHeader, err := r.FormFile(k)
		if err != nil {
			fmt.Println("err:", err)
			return
		}
		dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			log.Fatal(err)
		}

		defer file.Close()
		f, err := os.OpenFile(dir+"/public/"+fileHeader.Filename, os.O_WRONLY|os.O_CREATE, 0666)
		//
		//if err != nil {
		//	//return "", err
		//}

		// Copy the file to the destination path
		io.Copy(f, file)

	}

}

func download(writer http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("web/download.tmpl")
	if err != nil {
		fmt.Println("create template failed, err:", err)
		return
	}

	files, err := ioutil.ReadDir("public/")
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		fmt.Println(file.Name(), file.IsDir())
	}

	h := r.Host
	tmpl.Execute(writer, h)
}

func main() {
	//openBrowser("http://192.168.124.2:9876")

	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {

		path, err := os.Executable()
		if err != nil {
			log.Println(err)
		}
		exPath := filepath.Dir(path)
		fmt.Println("current path:", exPath)

		tmpl, err := template.ParseFiles(exPath + "/web/index.tmpl")
		if err != nil {
			fmt.Println("create template failed, err:", err)
			return
		}
		h := request.Host
		tmpl.Execute(writer, h)
	})

	http.HandleFunc("/upload", upload)
	http.HandleFunc("/download", download)

	err := http.ListenAndServe(":9876", nil)
	if err != nil {
		fmt.Println("HTTP server failed,err:", err)
		return
	}
}
