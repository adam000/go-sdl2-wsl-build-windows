package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type sdlPackage struct {
	name            string
	baseUrl         string
	dllName         string
	devName         string
	expandedDevName string
	version         string
}

var packages []sdlPackage = []sdlPackage{
	sdlPackage{
		name:            `SDL`,
		baseUrl:         `https://www.libsdl.org/release/`,
		dllName:         `SDL2-%s-win32-x64.zip`,
		devName:         `SDL2-devel-%s-mingw.tar.gz`,
		expandedDevName: `SDL2-%s`,
		version:         `2.0.8`,
	},
	sdlPackage{
		name:            `SDL_image`,
		baseUrl:         `https://www.libsdl.org/projects/SDL_image/release/`,
		dllName:         `SDL2_image-%s-win32-x64.zip`,
		devName:         `SDL2_image-devel-%s-mingw.tar.gz`,
		expandedDevName: `SDL2_image-%s`,
		version:         `2.0.3`,
	},
	sdlPackage{
		name:            `SDL_ttf`,
		baseUrl:         `https://www.libsdl.org/projects/SDL_ttf/release/`,
		dllName:         `SDL2_ttf-%s-win32-x64.zip`,
		devName:         `SDL2_ttf-devel-%s-mingw.tar.gz`,
		expandedDevName: `SDL2_ttf-%s`,
		version:         `2.0.14`,
	},
}

var vendorDir string = `vendor/sdl`

func main() {
	// Make out dir if necessary
	if err := os.MkdirAll("out/win64", os.FileMode(0777)|os.ModeDir); err != nil {
		panic(fmt.Errorf("Creating out dir: %v", err))
	}

	// Set working directory to `vendorDir`
	mkcd(vendorDir)

	// Process the packages specified.
	for _, pkg := range packages {
		mkcd(pkg.name)

		dllFullName := fmt.Sprintf(pkg.dllName, pkg.version)
		devFullName := fmt.Sprintf(pkg.devName, pkg.version)
		expandedFullName := fmt.Sprintf(pkg.expandedDevName, pkg.version)

		// Resolve URLs.
		dllFullUrl := resolveUrl(pkg.baseUrl, dllFullName)
		devFullUrl := resolveUrl(pkg.baseUrl, devFullName)

		// If the files don't exist, download them.
		downloadIfNeeded(dllFullName, dllFullUrl.String())
		downloadIfNeeded(devFullName, devFullUrl.String())

		// Expand the archives.
		expandArchive(dllFullName, pkg.expandedDevName)
		expandArchive(devFullName, pkg.expandedDevName)

		// Copy DLL to output directory in preparation.
		{
			log.Println("Copying dlls to output directory...")
			parameters := []string{"-f"}
			dlls, _ := filepath.Glob("*.dll")
			parameters = append(parameters, dlls...)
			parameters = append(parameters, "../../../out/win64")

			cmd := exec.Command("cp", parameters...)
			if output, err := cmd.CombinedOutput(); err != nil {
				log.Println(string(output))
				panic(fmt.Errorf("Copying dll to output directory: %v", err))
			}
		}

		// Copy / replace development package into the $PATH.
		placeDevelopmentSubdirs(expandedFullName)

		// Go back to where we were before, for the next iteration.
		os.Chdir("..")
	}
}

func resolveUrl(baseName, fileName string) *url.URL {
	fileUrl, err := url.Parse(fileName)
	if err != nil {
		panic(fmt.Errorf("Could not parse dev pkg full name (%s) into URL: %v", fileName, err))
	}
	baseUrl, err := url.Parse(baseName)
	if err != nil {
		panic(fmt.Errorf("Could not parse baseURL string (%s) into URL: %v", baseName, err))
	}
	return baseUrl.ResolveReference(fileUrl)
}

// If needed, make a directory and then change the current directory to there.
// In bash, this would be something like `mkdir -p $DIR && cd $DIR`.
func mkcd(dir string) {
	if err := os.MkdirAll(dir, os.FileMode(0777)|os.ModeDir); err != nil {
		panic(fmt.Errorf("Creating directory structure %s: %v", dir, err))
	}

	if err := os.Chdir(dir); err != nil {
		panic(fmt.Errorf("Creating directory structure %s: %v", dir, err))
	}
}

/*
// Copy a directory in $CWD (`from`) to another location (`to`), replacing it at its destination if necessary.
func replaceDir(from, to string) {
	destination := filepath.Join(to, from)
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		os.RemoveAll(destination)
	}

	copyDir(from, to)
}
*/

// TODO if you use this somewhere else, take out the sudo
func copyDir(from, to string) {
	cmd := exec.Command("sudo", "cp", "-r", from, to)
	//log.Printf("Progress: cp -r %s %s", from, to)
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		panic(fmt.Errorf("Copying %s to %s: %v", from, to, err))
	}
}

func placeDevelopmentSubdirs(expandedName string) {
	log.Println("Placing development files into /usr subdirs...")
	subdirs := []string{"bin", "include", "lib", "share"}

	os.Chdir(expandedName)
	os.Chdir("x86_64-w64-mingw32")
	defer os.Chdir("../..")

	for _, subdir := range subdirs {
		destination := filepath.Join("/usr/x86_64-w64-mingw32", subdir)
		if _, err := os.Stat(destination); os.IsNotExist(err) {
			if err := os.MkdirAll(destination, os.FileMode(0777)|os.ModeDir); err != nil {
				panic(fmt.Errorf("Creating %s dir: %v", destination, err))
			}
		}

		if _, err := os.Stat(subdir); !os.IsNotExist(err) {
			os.Chdir(subdir)
			filesOrDirs, _ := ioutil.ReadDir(".")
			for _, fileOrDir := range filesOrDirs {
				fileName := fileOrDir.Name()
				copyDir(fileName, destination)
			}
			os.Chdir("..")
		}
	}
}

func expandArchive(fileName, devFolder string) {
	// Make sure it ends with either .zip or .tar.gz, since those are the two types we know.
	log.Printf("Expanding %s", fileName)
	switch {
	case strings.HasSuffix(fileName, ".zip"):
		// Call `unzip -o $fileName`
		cmd := exec.Command("unzip", "-o", fileName)
		if err := cmd.Run(); err != nil {
			panic(fmt.Errorf("Unzipping %s: %v", fileName, err))
		}
	case strings.HasSuffix(fileName, ".tar.gz"):
		// Remove directory if it already exists.
		if _, err := os.Stat(devFolder); !os.IsNotExist(err) {
			os.RemoveAll(devFolder)
		}
		// Call `tar -xzf $fileName`
		cmd := exec.Command("tar", "-xzf", fileName)
		if err := cmd.Run(); err != nil {
			panic(fmt.Errorf("Untaring %s: %v", fileName, err))
		}
	default:
		panic(fmt.Errorf("Don't know how to expand archive %s", fileName))
	}
}

func downloadIfNeeded(localFileName, downloadUrl string) {
	_, err := os.Stat(localFileName)
	if os.IsNotExist(err) {
		log.Printf("Downloading %s", downloadUrl)
		cmd := exec.Command("wget", "--quiet", downloadUrl)
		if err := cmd.Run(); err != nil {
			panic(fmt.Errorf("Downloading %s: %v", downloadUrl, err))
		}
	}
}
