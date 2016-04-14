package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func dofail(err error, reader *bufio.Reader) {
	if err == nil {
		return
	}

	fmt.Printf("Failed: %s", err)
	reader.ReadLine()
	os.Exit(1)
}

func main() {
	doLock := true
	filename := ""
	pwd := ""

	reader := bufio.NewReader(os.Stdin)

	if len(os.Args) > 1 {
		finfo, err := os.Stat(os.Args[1])
		dofail(err, reader)
		filename = os.Args[1]
		if !finfo.IsDir() && filepath.Ext(filename) == ".locked" {
			doLock = false
		}

		if len(os.Args) > 2 {
			pwd = os.Args[2]
		}
	} else {
		fmt.Printf("(L)ock or (U)nlock?")
		c, _, _ := reader.ReadLine()

		if c[0] == 'l' {
			fmt.Printf("Lock Dir Name?")
			line, _, _ := reader.ReadLine()
			filename = string(line)
		} else {
			fmt.Printf("Lock File Name?")
			line, _, _ := reader.ReadLine()
			filename = string(line)
			doLock = false
		}
	}

	if pwd == "" {
		fmt.Printf("Password?\n")
		line, _, _ := reader.ReadLine()
		pwd = string(line)
	}
	sum := sha256.Sum256([]byte(pwd))

	if doLock {
		fmt.Printf("Encrypting: %s\n", filename)
		data, _ := doZip(filename)
		encryptData, err := encrypt(sum[:32], data)
		dofail(err, reader)
		ioutil.WriteFile(filepath.Clean(filename)+".locked", encryptData, 0644)
		os.RemoveAll(filename)
		fmt.Printf("Done\n")
	} else {
		fmt.Printf("Decrypting: %s\n", filename)
		encdata, err := ioutil.ReadFile(filename)
		dofail(err, reader)
		decrypted, err := decrypt(sum[:32], encdata)
		dofail(err, reader)

		err = doUnzip(strings.Split(string(filename), ".")[0], decrypted)
		dofail(err, reader)
		os.RemoveAll(filename)
	}
}

func doUnzip(target string, locked []byte) error {
	reader, err := zip.NewReader(bytes.NewReader(locked), int64(len(locked)))
	if err != nil {
		return err
	}

	if err := os.MkdirAll(target, 0755); err != nil {
		return err
	}

	for _, file := range reader.File {
		path := filepath.Join(target, file.Name)
		if file.FileInfo().IsDir() {
			os.MkdirAll(path, file.Mode())
			continue
		}

		fileReader, err := file.Open()
		if err != nil {
			return err
		}

		targetFile, err := os.Create(path)
		if err != nil {
			return err
		}

		if _, err := io.Copy(targetFile, fileReader); err != nil {
			return err
		}
		fileReader.Close()
		targetFile.Close()
	}

	return nil
}

func doZip(dir string) ([]byte, error) {
	zipfile := &bytes.Buffer{}

	archive := zip.NewWriter(zipfile)

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		// TODO: walk the tree!
		if info.IsDir() {
			return nil
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(writer, file)
		return nil
	})
	archive.Close()
	return zipfile.Bytes(), nil
}
