package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/fd/heroku-utils/pkg/xheroku"
	"gopkg.in/alecthomas/kingpin.v2"
	"limbo.services/version"
)

func main() {
	err := run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		inputTar string
		appName  string
	)

	app := kingpin.New("x-heroku", "Heroku utilities").Version(version.Get().String()).Author(version.Get().ReleasedBy)

	releaseCmd := app.Command("release", "Release an app")
	releaseCmd.Flag("input", "Tar archive to use").Short('i').Default("-").PlaceHolder("FILE").StringVar(&inputTar)
	releaseCmd.Arg("application", "Name of the application").Required().StringVar(&appName)

	switch kingpin.MustParse(app.Parse(os.Args[1:])) {

	case releaseCmd.FullCommand():
		r, err := openStream(inputTar)
		if err != nil {
			return err
		}

		slugID, err := xheroku.CreateSlug(r, appName)
		if err != nil {
			return err
		}

		err = xheroku.CreateRelease(appName, slugID)
		if err != nil {
			return err
		}

	}

	return nil
}

const stdio = "-"

func openStream(name string) (io.Reader, error) {
	if name == stdio {
		return os.Stdin, nil
	}
	return os.Open(name)
}

func putStream(name string, buf *bytes.Buffer) error {
	if name == stdio {
		_, err := io.Copy(os.Stdout, buf)
		return err
	}
	return ioutil.WriteFile(name, buf.Bytes(), 0644)
}
