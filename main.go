package main

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/melbahja/goph"
)

type Helper struct{}

type Bot struct {
	Owner   string `json:"owner"`
	Address string `json:"address"`
	Active  bool   `json:"active"`
}

var (
	h              Helper
	wg             sync.WaitGroup
	newHosts       = "new.host"
	activeBots     = "activebots.host"
	disactiveHosts = "disactive.host"
	statusfile     = "statusb.json"
	clientsName    = "clients.name"
	botLine        = "testBot"
)

// TODO create func thar return root path depade on os
func getRootPath() string {
	// do more
	return "/root/saverbot/"
}

//  zipfile.zip and clientName
func (Helper) remoteZip(cli *goph.Client, dir string) error {
	// zip the client bot app
	// zip -r test.zip testbot
	cmd, err := cli.Command("zip", "-r", dir+".zip", dir)
	if err != nil {
		return err
	}
	err = cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

// download zips and downloads remote dir
func (Helper) download(addr string) error {
	sshcli, err := goph.NewUnknown("root", addr /*"139.162.100.216"*/, goph.Password(h.getPass()))
	if err != nil {
		return errors.New("new connect err" + err.Error())
	}
	defer sshcli.Close()

	// get all remote file|dir
	dirs, err := h.ls(addr)
	if err != nil {
		return errors.New("remote ls err: " + err.Error())
	}

	for _, dirbot := range dirs {

		if strings.HasSuffix(dirbot, "-bot") {

			err = h.remoteZip(sshcli, dirbot)
			if err != nil {
				return errors.New("remote zip err: " + err.Error())
			}

			err = sshcli.Download("/root/"+dirbot+".zip", dirbot+".zip")
			if err != nil {
				return errors.New("download" + err.Error())
			}
		}
	}

	return nil
}

// run
func main() {

	for {
		hosts, err := h.load(activeBots)
		if err != nil {
			fmt.Println(err)
			goto here
		}

		for _, host := range hosts {

			fmt.Println("downloading from:", host)

			err := h.download(host)
			if err != nil {
				fmt.Printf("err with host %s: \n %s\n", host, err.Error())
			}
			fmt.Printf("Done\n")

		}
	here:
		time.Sleep(time.Minute)

	}
}

// deploy deploy client-bot.zip to client host
func (Helper) deploy(clientBot, hostBot string) error {
	sshClient, err := goph.NewUnknown("root", hostBot, goph.Password(h.getPass()))
	if err != nil {
		return err
	}

	clientBot = clientBot + "-bot.zip"

	fmt.Println(clientBot)
	err = sshClient.Upload(clientBot, clientBot)
	if err != nil {
		return err
	}

	return nil
}

// to scure app read pass form seprite file
func (Helper) getPass() string {
	data, err := ioutil.ReadFile(".mypass")
	if err != nil {
		return err.Error()
	}
	psw := string(data)
	return psw[:len(psw)-1]
}

// ls list remote file/dir
func (Helper) ls(addr string) ([]string, error) {
	sshcli, err := goph.NewUnknown("root", addr /*"139.162.100.216"*/, goph.Password(h.getPass()))
	if err != nil {
		return nil, err
	}
	defer sshcli.Close()

	out, err := sshcli.Run("ls")
	if err != nil {
		return nil, err
	}
	dirs := strings.Split(string(out), "\n")
	return dirs, nil

}

// File copies a single file from src to dst
func (Helper) copyLocalFile(src, dst string) error {
	var err error
	var srcfd *os.File
	var dstfd *os.File
	var srcinfo os.FileInfo

	if srcfd, err = os.Open(src); err != nil {
		return err
	}
	defer srcfd.Close()

	if dstfd, err = os.Create(dst); err != nil {
		return err
	}
	defer dstfd.Close()

	if _, err = io.Copy(dstfd, srcfd); err != nil {
		return err
	}
	if srcinfo, err = os.Stat(src); err != nil {
		return err
	}
	return os.Chmod(dst, srcinfo.Mode())
}

// copyDir copies local botLine directory
// this is copies a whole directory recursively
func (Helper) copyLocalDir(src string, dst string) error {
	dst = dst + "-bot"
	var err error
	var fds []os.FileInfo
	var srcinfo os.FileInfo

	if srcinfo, err = os.Stat(src); err != nil {
		fmt.Println("err: os.Stat")
		return err
	}

	if err = os.MkdirAll(dst, srcinfo.Mode()); err != nil {
		fmt.Println("err: os.MakeAll")
		return err
	}

	if fds, err = ioutil.ReadDir(src); err != nil {
		fmt.Println("err: ioutil.ReadDir")
		return err
	}
	for _, fd := range fds {
		srcfp := path.Join(src, fd.Name())
		dstfp := path.Join(dst, fd.Name())

		if fd.IsDir() {
			if err = h.copyLocalDir(srcfp, dstfp); err != nil {

				fmt.Println("err: recoursive 1")
				fmt.Println(err)
			}
		} else {
			if err = h.copyLocalFile(srcfp, dstfp); err != nil {
				fmt.Println("err: recoursive 2")
				fmt.Println(err)
			}
		}
	}

	// creat a new file that containe client info,
	clientInfo, err := os.Create(dst + "/" + dst + ".info")
	if err != nil {

		fmt.Println("creat file info when copping dir")
		return err
	}
	defer clientInfo.Close()
	clientInfo.WriteString(dst)

	return nil
}

// clientInStatus if client or host are in status
func (h Helper) clientInStatus(owner string, bots *[]Bot) bool {
	for _, bot := range *bots {
		if owner == bot.Owner {
			return true
		}
	}
	return false
}

// InStatus if client or host are in status
func (h Helper) hostInStatus(host string, bots *[]Bot) bool {
	for _, bot := range *bots {
		if host == bot.Address {
			return true
		}
	}
	return false
}

// updateStatusf update status file
func (Helper) updateStatusf(data []byte) error {
	if err := os.WriteFile(statusfile, []byte(data), 0644); err != nil {
		return (err)
	}
	return nil
}

// return list of bots type
func (Helper) loadStatus() ([]Bot, error) {

	bots := make([]Bot, 5)
	data, err := ioutil.ReadFile(statusfile)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, &bots)
	if err != nil {
		return nil, err
	}
	return bots, nil
}

// removeItem remove Item string from list and return new list
func (Helper) removeItem(item string, list []string) []string {
	newList := make([]string, 0)
	for _, v := range list {
		if item != v {
			newList = append(newList, v)
		}
	}
	return newList
}

// check if host is active ?
func (Helper) isHostActive(host string) bool {
	client, err := goph.NewUnknown("root", host, goph.Password(h.getPass()))
	if err != nil {
		return false
	}
	client.Close()
	return true
}

// writeData updates/rewrites data into file
func (Helper) update(file string, list []string) error {
	data := ""
	for _, item := range list {
		data += item + "\n"
	}
	err := os.WriteFile(file, []byte(data), 0644)
	if err != nil {
		log.Println(err)
	}
	return err
}

// load loads file and return hosts address as []stirng
func (Helper) load(file string) ([]string, error) {

	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	hs := strings.Split(string(data), "\n")

	list := make([]string, 0)

	for _, v := range hs {

		h := strings.Replace(v, " ", "", -1)
		if len(h) < 3 {
			continue
		}
		list = append(list, h)
	}

	return h.unique(list), nil
}

// appendAddress appends new address to addressfile
func (Helper) appendAddr(file, data string) {
	f, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
	}
	defer f.Close()
	if _, err := f.WriteString(data + "\n"); err != nil {
		log.Println(err)
	}
}

// filterList make list unique
func (Helper) unique(list []string) []string {
	mp := make(map[string]bool)
	for _, h := range list {
		mp[h] = true
	}
	ulist := make([]string, 0)
	for h := range mp {
		if h == "" {
			break
		}
		ulist = append(ulist, h)
	}
	return ulist
}

// activeHosts filter hosts and return just active hostes
func (Helper) activeHosts(bots []Bot) []Bot {

	activeBots := make([]Bot, 0)
	for _, bot := range bots {
		if bot.Active {
			activeBots = append(activeBots, bot)
		} else {
			h.appendAddr("disactive.host", bot.Address)
		}
	}
	return activeBots
}

// send exitbot
func (Helper) sendExit(address string) {
	resp, err := http.Get("http://" + address + "/exit")
	if err != nil {
		log.Fatal("Error getting response. ", err)
	}
	defer resp.Body.Close()

	// Read body from response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Error reading response. ", err)
	}

	fmt.Printf("body is : %s\n", body)
}

// may be not need this func
// Copies a file. and rename to name with .cp saffix
func (Helper) copyFile(src string) error {
	// Open the source file for reading
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	// Open the destination file for writing
	d, err := os.Create(src + ".cp")
	if err != nil {
		return err
	}

	// Copy the contents of the source file into the destination file
	if _, err := io.Copy(d, source); err != nil {
		d.Close()
		return err
	}

	// Return any errors that result from closing the destination file
	// Will return nil if no errors occurred
	return d.Close()
}

// TODO test localzip function
//  zipfile.zip and clientName
func (Helper) zipLocalDir(source string) error {
	// 1. Create a ZIP file and zip.Writer
	f, err := os.Create(source + "-bot.zip")
	if err != nil {
		return err
	}
	defer f.Close()

	writer := zip.NewWriter(f)
	defer writer.Close()

	// 2. Go through all the files of the source
	return filepath.Walk(source+"-bot", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 3. Create a local file header
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// set compression
		header.Method = zip.Deflate

		// 4. Set relative path of a file as the header name
		header.Name, err = filepath.Rel(filepath.Dir(source), path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			header.Name += "/"
		}

		// 5. Create writer for the file header and save content of the file
		headerWriter, err := writer.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(headerWriter, f)
		if err != nil {
			return err
		}
		return nil

	})
}

// checkErr check error if exeste and close program
func checkErr(info string, err error) {
	if err != nil {
		log.Println(info, err)
		os.Exit(0)
	}
}
