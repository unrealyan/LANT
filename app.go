package main

import (
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

type file struct {
	Name string `json:"name"`
	File os.File
}

func getLocalIpV4() string {
	inters, err := net.Interfaces()
	if err != nil {
		panic(err)
	}
	for _, inter := range inters {
		// 判断网卡是否开启，过滤本地环回接口
		if inter.Flags&net.FlagUp != 0 && !strings.HasPrefix(inter.Name, "lo") {
			// 获取网卡下所有的地址
			addrs, err := inter.Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addrs {
				if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
					//判断是否存在IPV4 IP 如果没有过滤
					if ipnet.IP.To4() != nil {
						return ipnet.IP.String()
					}
				}
			}
		}
	}
	return ""
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

func apiUpload(writer http.ResponseWriter, r *http.Request) {

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

func Upload(writer http.ResponseWriter, r *http.Request) {
	//path, err := os.Executable()
	exPath, err := getCurrentAbPath()
	if err != nil {
		log.Println(err)
	}
	//exPath := filepath.Dir(_path)
	fmt.Println("current path:", exPath)

	tmpl, err := template.ParseFiles(exPath + "/web/upload.tmpl")
	if err != nil {
		fmt.Println("create template failed, err:", err)
		return
	}
	h := r.Host
	tmpl.Execute(writer, h)
}

// 最终方案-全兼容
func getCurrentAbPath() (string, error) {
	dir := getCurrentAbPathByExecutable()
	tmpDir, _ := filepath.EvalSymlinks(os.TempDir())
	if strings.Contains(dir, tmpDir) {
		return getCurrentAbPathByCaller(), nil
	}
	return dir, nil
}

// 获取当前执行文件绝对路径
func getCurrentAbPathByExecutable() string {
	exePath, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	res, _ := filepath.EvalSymlinks(filepath.Dir(exePath))
	return res
}

// 获取当前执行文件绝对路径（go run）
func getCurrentAbPathByCaller() string {
	var abPath string
	_, filename, _, ok := runtime.Caller(0)
	if ok {
		abPath = path.Dir(filename)
	}
	return abPath
}

func SendJqueryJs(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.RequestURI)
	exPath, err := getCurrentAbPath()
	if err != nil {
		log.Println(err)
	}
	data, err := ioutil.ReadFile(exPath + "/web/qrcode.js")
	if err != nil {
		http.Error(w, "Couldn't read file", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Write(data)
}

func main() {

	//dir, er := filepath.Abs(filepath.Dir(os.Args[0]))
	dir, er := getCurrentAbPath()
	if er != nil {
		log.Fatal(er)
	}
	if _, err := os.Stat(dir + "/public/"); os.IsNotExist(err) {
		err = os.Mkdir(dir+"/public/", 0777)
		// TODO: handle error
	}
	ip := getLocalIpV4()
	openBrowser("http://" + ip + ":9876")

	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {

		//path, err := os.Executable()
		exPath, err := getCurrentAbPath()
		if err != nil {
			log.Println(err)
		}
		//exPath := filepath.Dir(_path)
		fmt.Println("current path:", exPath)

		tmpl, err := template.ParseFiles(exPath + "/web/index.tmpl")
		if err != nil {
			fmt.Println("create template failed, err:", err)
			return
		}
		h := request.Host
		tmpl.Execute(writer, "http://"+h+"/upload")
	})

	http.HandleFunc("/api/upload", apiUpload)
	http.HandleFunc("/upload", Upload)
	http.HandleFunc("/download", download)
	http.HandleFunc("/qrcode.js", SendJqueryJs)

	err := http.ListenAndServe(":9876", nil)
	if err != nil {
		fmt.Println("HTTP server failed,err:", err)
		return
	}
}
