package xheroku

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
)

type releaseOptions struct {
	Slug string `json:"slug"`
}

func CreateRelease(appID, slugID string) error {
	options := releaseOptions{
		Slug: slugID,
	}

	payload, err := json.Marshal(&options)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", "https://api.heroku.com/apps/"+appID+"/releases", bytes.NewReader(payload))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.heroku+json; version=3")
	req.Header.Set("Authorization", "Bearer "+os.Getenv("HEROKU_API_KEY"))

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()
	defer io.Copy(ioutil.Discard, res.Body)

	if res.StatusCode/100 != 2 {
		return fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	return nil
}
