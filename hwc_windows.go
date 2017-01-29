// +build windows
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"text/template"
	"unsafe"
)

var (
	hwebcore, _        = syscall.LoadLibrary(os.ExpandEnv(`${windir}\\\system32\inetsrv\hwebcore.dll`))
	webCoreActivate, _ = syscall.GetProcAddress(hwebcore, "WebCoreActivate")
	webCoreShutdown, _ = syscall.GetProcAddress(hwebcore, "WebCoreShutdown")
	appRootPath        string
	ErrMissingPortEnv  = errors.New("Missing PORT environment variable")
)

func WebCoreActivate(appHostConfig, rootWebConfig, instanceName string) error {
	var nargs uintptr = 3
	r1, _, callErr := syscall.Syscall(uintptr(webCoreActivate),
		nargs,
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(appHostConfig))),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(rootWebConfig))),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(instanceName))))

	fmt.Println(r1, " - ", callErr)
	if callErr != 0 {
		return callErr
	}
	fmt.Println("Server Started")
	return nil
}
func WebCoreShutdown(immediate int) error {
	var nargs uintptr = 1
	r1, _, callErr := syscall.Syscall(uintptr(webCoreShutdown),
		nargs, uintptr(unsafe.Pointer(&immediate)), 0, 0)

	fmt.Println(r1, " - ", callErr)
	if callErr != 0 {
		return callErr
	}
	fmt.Println("Server Shutdown")
	return nil
}

type App struct {
	Port     int
	RootPath string
}

func ahcConfig(config App) error {
	ahcFile, err := os.Create("C:\\applicationhost.config")
	if err != nil {
		return err
	}
	defer ahcFile.Close()
	var tmpl = template.Must(template.New("applicationhost").Parse(ApplicationHostConfig))
	if err := tmpl.Execute(ahcFile, config); err != nil {
		return err
	}
	return nil
}
func main() {
	flag.Parse()
	defer syscall.FreeLibrary(hwebcore)

	if os.Getenv("PORT") == "" {
		checkErr(ErrMissingPortEnv)
	}
	port, err := strconv.Atoi(os.Getenv("PORT"))
	checkErr(err)
	rootPath, err := filepath.Abs(appRootPath)
	checkErr(err)

	app := App{
		Port:     port,
		RootPath: rootPath,
	}

	checkErr(ahcConfig(app))

	checkErr(WebCoreActivate(
		"c:\\applicationhost.config",
		"c:\\web.config",
		"aj01"))

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	s := <-c
	checkErr(WebCoreShutdown(1))
	fmt.Println("Got signal:", s)
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
func init() {
	flag.StringVar(&appRootPath, "appRootPath", ".", "app web root path")
}
