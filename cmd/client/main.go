package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jchv/go-webview2"
)

func main() {
	// Read server URL from server.txt
	exePath, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	dir := filepath.Dir(exePath)
	serverURLBytes, err := ioutil.ReadFile(filepath.Join(dir, "server.txt"))
	if err != nil {
		log.Println("Could not read server.txt, defaulting to http://localhost:8080")
		serverURLBytes = []byte("http://localhost:8080")
	}
	url := strings.TrimSpace(string(serverURLBytes))

	// Create WebView
	w := webview2.New(false)
	defer w.Destroy()
	w.SetTitle("HwpPdfSearcher")
	w.SetSize(1024, 768, webview2.HintNone)

	// Bind Go function to JavaScript
	w.Bind("openFile", func(path string) {
		log.Println("Opening file:", path)
		cmd := exec.Command("cmd", "/c", "start", "", path)
		if err := cmd.Start(); err != nil {
			log.Println("Failed to open file:", err)
		}
	})

	w.Navigate(url)
	w.Run()
}
