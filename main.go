package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var extensions = map[string]string{
	".go":           "//",
	".js":           "//",
	".proto":        "//",
	".sql":          "--",
	".gitignore":    "#",
	".dockerignore": "#",
	".helmignore":   "#",
	".tf":           "#",
	".tfvars":       "#",
	".bashrc":       "#",
	"Dockerfile":    "#",
	"Makefile":      "#",
}

func addLicenseInFolder(
	ctx context.Context,
	report *Stat,
	path string,
	lines []string,
) {
	err := filepath.Walk(path, func(fpath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		addLicenseInFile(ctx, report, fpath, lines)
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}

func addLicenseInFile(ctx context.Context,
	report *Stat,
	path string,
	lines []string,
) error {
	cType, ok := extensions[filepath.Ext(path)]
	if !ok {
		if cType, ok = extensions[filepath.Base(path)]; !ok {
			return nil
		}
	}
	if strings.Contains(path, "node_modules/") ||
		strings.Contains(path, ".min.") {
		return nil
	}

	// ok. we match the expression
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	str := string(b)
	flines := strings.Split(str, "\n")
	firstLines := ""
	if len(flines) > 5 {
		firstLines = strings.Join(flines[0:5], "\n")
	} else {
		firstLines = strings.Join(flines, "\n")
	}
	comments := ""
	for _, l := range lines {
		comments += cType + " " + l + "\n"
	}
	comments += "\n"

	if strings.Contains(firstLines, "Copyright") {
		fmt.Println("[ALREADY]     " + path)
		report.Skipped++
	} else if strings.Contains(firstLines, "DO NOT EDIT") {
		fmt.Println("[DO NOT EDIT] " + path)
		report.DoNotModify++
	} else {
		fmt.Println("[INSERTED]    " + path)
		ioutil.WriteFile(path, []byte(comments+str), 0644)
		report.Added++
	}
	return nil
}

func downloadLicenseMarkdown(ctx context.Context, URL string, outFile string) error {
	fmt.Println("Downloading from " + URL)

	// Create the file
	resp, err := http.Get(URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(outFile)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}
	return nil
}

func hasLicenseFile(path string) bool {
	if _, err := os.Stat(path + "/LICENSE"); !os.IsNotExist(err) {
		return true
	}
	if _, err := os.Stat(path + "/LICENSE.md"); !os.IsNotExist(err) {
		return true
	}
	return false
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Path or file name is required as the first argument")
	}
	path := os.Args[1]
	snippet := os.Getenv("LICENSE_SNIPPET")
	if snippet == "" {
		log.Fatal("Please set up LICENSE_SNIPPET env var")
	}
	fi, err := os.Stat(path)
	if err != nil {
		log.Fatal(err)
	}
	lines := strings.Split(snippet, "\\n")

	ctx := context.Background()
	report := NewStat()
	switch mode := fi.Mode(); {
	case mode.IsDir():
		URL := os.Getenv("LICENSE_URL")
		if URL != "" && !hasLicenseFile(path) {
			errD := downloadLicenseMarkdown(ctx, URL, path+"/LICENSE")
			if errD != nil {
				log.Fatal(errD)
			}
		}
		addLicenseInFolder(ctx, report, path, lines)
	case mode.IsRegular():
		addLicenseInFile(ctx, report, path, lines)
	}

	fmt.Println(report.String())
}
