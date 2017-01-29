// +build windows

package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"text/template"
	"unsafe"

	"github.com/docker/distribution/uuid"
)

var (
	appRootPath       string
	ErrMissingPortEnv = errors.New("Missing PORT environment variable")
)

type webCore struct {
	activated bool
	handle    syscall.Handle
}

func newWebCore() (*webCore, error) {
	hwebcore, err := syscall.LoadLibrary(os.ExpandEnv(`${windir}\\\system32\inetsrv\hwebcore.dll`))
	if err != nil {
		return nil, err
	}
	return &webCore{
		activated: false,
		handle:    hwebcore,
	}, nil
}

func (w *webCore) Activate(appHostConfig, rootWebConfig, instanceName string) error {
	if !w.activated {
		webCoreActivate, err := syscall.GetProcAddress(w.handle, "WebCoreActivate")
		if err != nil {
			return err
		}
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
		w.activated = true
	}
	return nil
}
func (w *webCore) Shutdown(immediate int) error {
	if w.activated {
		webCoreShutdown, err := syscall.GetProcAddress(w.handle, "WebCoreShutdown")
		if err != nil {
			return err
		}
		var nargs uintptr = 1
		r1, _, callErr := syscall.Syscall(uintptr(webCoreShutdown),
			nargs, uintptr(unsafe.Pointer(&immediate)), 0, 0)

		fmt.Println(r1, " - ", callErr)
		if callErr != 0 {
			return callErr
		}
		fmt.Println("Server Shutdown")
	}
	return nil
}

type App struct {
	Port                  int
	RootPath              string
	ApplicationHostConfig string
	AspnetConfig          string
	WebConfig             string
}

func (a App) applicationHostConfig() error {
	file, err := os.Create(a.ApplicationHostConfig)
	if err != nil {
		return err
	}
	defer file.Close()
	var tmpl = template.Must(template.New("applicationhost").Parse(ApplicationHostConfig))
	if err := tmpl.Execute(file, a); err != nil {
		return err
	}
	return nil
}
func (a App) aspnetConfig() error {
	file, err := os.Create(a.AspnetConfig)
	if err != nil {
		return err
	}
	defer file.Close()
	var tmpl = template.Must(template.New("aspnet").Parse(AspnetConfig))
	if err := tmpl.Execute(file, a); err != nil {
		return err
	}
	return nil
}
func (a App) webConfig() error {
	rootWebConfig := os.ExpandEnv(`${windir}\\\Microsoft.NET\Framework\v4.0.30319\Config\web.config`)
	in, err := os.Open(rootWebConfig)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(a.WebConfig)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	err = out.Close()
	if err != nil {
		return err
	}
	return nil
}
func (a *App) configure() error {
	dest := filepath.Join(a.RootPath, ".cloudfoundry", "hwc")
	err := os.MkdirAll(dest, 0700)
	if err != nil {
		return err
	}
	a.ApplicationHostConfig = filepath.Join(dest, "applicationhost.config")
	a.AspnetConfig = filepath.Join(dest, "aspnet.config")
	a.WebConfig = filepath.Join(dest, "web.config")
	err = a.applicationHostConfig()
	if err != nil {
		return err
	}
	err = a.aspnetConfig()
	if err != nil {
		return err
	}
	err = a.webConfig()
	if err != nil {
		return err
	}
	return nil
}
func main() {
	flag.Parse()

	wc, err := newWebCore()
	checkErr(err)
	defer syscall.FreeLibrary(wc.handle)

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
	checkErr(app.configure())

	checkErr(wc.Activate(
		app.ApplicationHostConfig,
		app.WebConfig,
		uuid.Generate().String()))

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	s := <-c
	checkErr(wc.Shutdown(1))
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
func init() {
	flag.StringVar(&appRootPath, "appRootPath", ".", "app web root path")
}
