package testHelpers

import (
	"gleaner/cmd/config"
	"gleaner/internal/projectpath"
	"net"
	"net/http"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

func ServeSampleConfigDir() (*http.Server, net.Listener, error) {
	dir := filepath.Join(projectpath.Root, "testHelpers", "sampleConfigs")

	fileServer := http.FileServer(http.Dir(dir))
	mux := http.NewServeMux()
	mux.Handle("/", fileServer)

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return nil, nil, err
	}

	server := &http.Server{
		Handler: mux,
	}

	go func() {
		log.Printf("Static file server running on %s", listener.Addr().String())
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Printf("Error serving: %v", err)
		}
	}()

	return server, listener, nil
}

// Create a config file with a specific url in the sources array replaced with a dynamic url value at runtime
func NewTempConfig(fileName, configDir string) (string, error) {
	conf, err := config.ReadGleanerConfig(fileName, configDir)
	if err != nil {
		return "", err
	}

	// Generate a temporary file path
	tempConfigPath := filepath.Join(configDir, "mockedGleanerConfig.yaml")

	// Write the modified configuration to the temporary file
	err = conf.WriteConfigAs(tempConfigPath)
	if err != nil {
		return "", err
	}

	return tempConfigPath, nil
}

// Update the value of the url in the sources array
func MutateYamlSourceUrl(configPath string, index int, url string) error {
	base := filepath.Base(configPath)
	dir := filepath.Dir(configPath)

	conf, err := config.ReadGleanerConfig(base, dir)
	if err != nil {
		return err
	}

	// Get the sources array
	// Retrieve the sources array
	var sources []map[string]interface{}
	if err := conf.UnmarshalKey("sources", &sources); err != nil {
		return err
	}

	// Update the url in the sources array
	sources[index]["url"] = url

	conf.Set("sources", sources)

	// Write the modified configuration to the temporary file
	err = conf.WriteConfigAs(configPath)
	if err != nil {
		return err
	}
	return nil

}
