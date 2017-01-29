package main_test

import (
	// "io/ioutil"

	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

const PORT = 43311

var _ = Describe("hwc", func() {
	var (
		err    error
		binary string
	)

	BeforeEach(func() {
		binary, err = Build("github.com/aminjam/hwc")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(binary)
	})

	sendCtrlBreak := func(s *Session) {
		d, err := syscall.LoadDLL("kernel32.dll")
		Expect(err).To(Succeed())
		p, err := d.FindProc("GenerateConsoleCtrlEvent")
		Expect(err).To(Succeed())
		r, _, err := p.Call(syscall.CTRL_BREAK_EVENT, uintptr(s.Command.Process.Pid))
		Expect(r).ToNot(Equal(0), fmt.Sprintf("GenerateConsoleCtrlEvent: %v\n", err))
	}

	startApp := func(name string) (*Session, error) {
		wd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		cmd := exec.Command(binary)
		cmd.Env = append(os.Environ(), fmt.Sprintf("PORT=%d", PORT))
		cmd.Dir = filepath.Join(wd, "fixtures", name)
		cmd.SysProcAttr = &syscall.SysProcAttr{
			CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
		}
		session, err := Start(cmd, GinkgoWriter, GinkgoWriter)
		if err != nil {
			return nil, err
		}
		Eventually(session).Should(Say("Server Started"))
		return session, err
	}
	Context("Given that I have ASP.NET MVC application", func() {
		var (
			session *Session
			err     error
		)
		BeforeEach(func() {
			session, err = startApp("nora")
			Expect(err).To(Succeed())
		})
		AfterEach(func() {
			sendCtrlBreak(session)
			Eventually(session, 10*time.Second).Should(Say("Server Shutdown"))
			<-session.Exited
		})

		It("runs it on the specified port", func() {
			url := fmt.Sprintf("http://localhost:%d", PORT)
			res, err := http.Get(url)
			Expect(err).To(Succeed())
			body, err := ioutil.ReadAll(res.Body)
			Expect(err).To(Succeed())
			Expect(string(body)).To(Equal(`"hello i am nora"`))
		})
	})
})
