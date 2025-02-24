package shapes

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"

	"gleaner/internal/common"
)

// Call the SHACL service container (or cloud instance) // TODO: service URL needs to be in the config file!
func shaclCallNG(url, dg, sg string) (string, error) {
	// datagraph, err := millerutils.JSONLDToTTL(dg, urlval)
	// if err != nil {
	// 	log.Printf("Error in the jsonld write... %v\n", err)
	// 	log.Printf("Nothing to do..   going home")
	// 	return 0
	// }

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	// writer.WriteField("datagraph", urlval)
	// writer.WriteField("shapegraph", sgkey)
	err := writer.WriteField("fmt", "nt")
	if err != nil {
		return "", err
	}

	//part, err := writer.CreateFormFile("datagraph", "datagraph")
	part, err := writer.CreateFormFile("dg", "datagraph")
	if err != nil {
		return "", err
	}
	_, err = io.Copy(part, strings.NewReader(dg))
	if err != nil {
		return "", err
	}

	//part, err = writer.CreateFormFile("shapegraph", "shapegraph")
	part, err = writer.CreateFormFile("sg", "shapegraph")
	if err != nil {
		return "", err
	}
	_, err = io.Copy(part, strings.NewReader(sg))
	if err != nil {
		return "", err
	}

	err = writer.Close()
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("User-Agent", "EarthCube_DataBot/1.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(b), err //  we will return the bytes count we write...
}

// DEPRECATED CODE BELOW..   will be replaced
// Call the SHACL service container (or cloud instance) // TODO: service URL needs to be in the config file!
func shaclTest(urlval, dg, sgkey, sg string, gb *common.Buffer) int {
	// datagraph, err := millerutils.JSONLDToTTL(dg, urlval)
	// if err != nil {
	// 	log.Printf("Error in the jsonld write... %v\n", err)
	// 	log.Printf("Nothing to do..   going home")
	// 	return 0
	// }

	url := "http://localhost:8080/uploader" // TODO this should be set in the config file
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	err := writer.WriteField("datagraph", urlval)
	if err != nil {
		log.Error(err)
	}
	err = writer.WriteField("shapegraph", sgkey)
	if err != nil {
		log.Error(err)
	}

	part, err := writer.CreateFormFile("datagraph", "datagraph")
	if err != nil {
		log.Error(err)
	}
	_, err = io.Copy(part, strings.NewReader(dg))
	if err != nil {
		log.Error(err)
	}

	part, err = writer.CreateFormFile("shapegraph", "shapegraph")
	if err != nil {
		log.Error(err)
	}
	_, err = io.Copy(part, strings.NewReader(sg))
	if err != nil {
		log.Error(err)
	}

	err = writer.Close()
	if err != nil {
		log.Error(err)
	}

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		log.Error(err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("User-Agent", "EarthCube_DataBot/1.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Error(err)
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error(err)
	}

	// write result to buffer
	len, err := gb.Write(b)
	if err != nil {
		log.Error("error in the buffer write...", err)
	}

	return len //  we will return the bytes count we write...
}
