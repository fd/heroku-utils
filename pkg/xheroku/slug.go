package xheroku

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

type slugInfo struct {
	ID                           string            `json:"id,omitempty"`
	CreatedAt                    *time.Time        `json:"created_at,omitempty"`
	UpdatedAt                    *time.Time        `json:"updated_at,omitempty"`
	BuildpackProvidedDescription *string           `json:"buildpack_provided_description,omitempty"`
	Checksum                     *string           `json:"checksum,omitempty"`
	Commit                       *string           `json:"commit,omitempty"`
	CommitDescription            *string           `json:"commit_description,omitempty"`
	ProcessTypes                 map[string]string `json:"process_types,omitempty"`
	Size                         int64             `json:"size,omitempty"`

	Stack *struct {
		ID   string `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"stack,omitempty"`

	Blob *struct {
		Method string `json:"method,omitempty"`
		URL    string `json:"url,omitempty"`
	} `json:"blob,omitempty"`
}

type SlugConfig struct {
	Commit                       string            `json:"commit,omitempty"`
	CommitDescription            string            `json:"commit_description,omitempty"`
	BuildpackProvidedDescription string            `json:"buildpack_provided_description,omitempty"`
	ProcessTypes                 map[string]string `json:"process_types,omitempty"`
	Stack                        string            `json:"stack,omitempty"`
}

type slugOptions struct {
	Checksum                     string            `json:"checksum,omitempty"`
	Commit                       string            `json:"commit,omitempty"`
	CommitDescription            string            `json:"commit_description,omitempty"`
	BuildpackProvidedDescription string            `json:"buildpack_provided_description,omitempty"`
	ProcessTypes                 map[string]string `json:"process_types,omitempty"`
	Stack                        string            `json:"stack,omitempty"`
}

func CreateSlug(r io.Reader, appID string) (string, error) {
	var (
		buf      bytes.Buffer
		checksum string
		config   *SlugConfig
		info     *slugInfo
	)

	{
		tr := tar.NewReader(r)
		sw := sha256.New()
		zw := gzip.NewWriter(io.MultiWriter(&buf, sw))
		tw := tar.NewWriter(zw)
		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return "", err
			}

			fmt.Fprintf(os.Stderr, "- %s\n", hdr.Name)
			hdr.Name = path.Join("/", hdr.Name)
			fmt.Fprintf(os.Stderr, "  %s\n", hdr.Name)
			if hdr.Name == "/.heroku.json" {
				err = json.NewDecoder(tr).Decode(&config)
				if err != nil {
					return "", err
				}

				_, err = io.Copy(ioutil.Discard, tr)
				if err != nil && err != io.EOF {
					return "", err
				}

				continue
			}

			hdr.Name = "./app" + hdr.Name
			if hdr.FileInfo().IsDir() {
				hdr.Name = strings.TrimSuffix(hdr.Name, "/") + "/"
			}
			fmt.Fprintf(os.Stderr, "  %s\n", hdr.Name)

			err = tw.WriteHeader(hdr)
			if err != nil {
				return "", err
			}

			_, err = io.Copy(tw, tr)
			if err != nil {
				return "", err
			}
		}

		err := tw.Close()
		if err != nil {
			return "", err
		}

		err = zw.Close()
		if err != nil {
			return "", err
		}

		checksum = "SHA256:" + hex.EncodeToString(sw.Sum(nil))
	}

	{
		options := slugOptions{
			Checksum:                     checksum,
			Commit:                       config.Commit,
			CommitDescription:            config.CommitDescription,
			BuildpackProvidedDescription: config.BuildpackProvidedDescription,
			ProcessTypes:                 config.ProcessTypes,
			Stack:                        config.Stack,
		}

		payload, err := json.Marshal(&options)
		if err != nil {
			return "", err
		}

		req, err := http.NewRequest("POST", "https://api.heroku.com/apps/"+appID+"/slugs", bytes.NewReader(payload))
		if err != nil {
			return "", err
		}

		fmt.Fprintf(os.Stderr, "payload: %s\n", payload)

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/vnd.heroku+json; version=3")
		req.Header.Set("Authorization", "Bearer "+os.Getenv("HEROKU_API_KEY"))

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", err
		}

		defer res.Body.Close()
		defer io.Copy(ioutil.Discard, res.Body)

		if res.StatusCode/100 != 2 {
			return "", fmt.Errorf("unexpected status code: %d", res.StatusCode)
		}

		err = json.NewDecoder(res.Body).Decode(&info)
		if err != nil {
			return "", err
		}
	}

	infoJSON, _ := json.Marshal(&info)
	fmt.Fprintf(os.Stderr, "info: %q\n", infoJSON)

	{
		req, err := http.NewRequest("PUT", info.Blob.URL, bytes.NewReader(buf.Bytes()))
		if err != nil {
			return "", fmt.Errorf("upload slug: %s", err)
		}

		req.Header.Set("Content-Type", "")

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("upload slug: %s", err)
		}

		defer res.Body.Close()
		defer io.Copy(ioutil.Discard, res.Body)

		if res.StatusCode/100 != 2 {
			return "", fmt.Errorf("upload slug: unexpected status code: %d", res.StatusCode)
		}
	}

	return info.ID, nil
}
