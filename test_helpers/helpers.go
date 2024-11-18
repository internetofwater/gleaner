package test_helpers

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	minioClient "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/minio"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Checks if every line in actual is present in expected, disregarding the order of lines.
// Useful for checking equivalence of nq files where line order doesn't matter
func AssertLinesMatchDisregardingOrder(expected string, actual string) bool {
	expectedLines := strings.Split(expected, "\n")
	actualLines := strings.Split(actual, "\n")

	// Create a hashmap to store the lines in expectedLines
	expectedMap := make(map[string]bool)
	for _, line := range expectedLines {
		expectedMap[line] = true
	}

	// Check if each line in actualLines is present in the hashmap
	// if it is remove it, so we can see if the hashmap is empty
	// at the end or if there are additional lines
	for _, line := range actualLines {
		if _, found := expectedMap[line]; found {
			delete(expectedMap, line)
		} else {
			return false
		}
	}

	return len(expectedMap) == 0
}

// Assert the number of objects in a minio subdir is expected
func AssertObjectCount(t *testing.T, mc *minioClient.Client, subDir string, expected int) {

	_, summoned, err := GetGleanerBucketObjects(mc, subDir)
	assert.NoError(t, err)
	assert.Equal(t, expected, len(summoned))

}

func RequireFilenameExists(t *testing.T, arr []minioClient.ObjectInfo, filename string) {
	for _, obj := range arr {
		if obj.Key == filename {
			return
		}
	}
	require.Fail(t, "filename not found")
}

func RequireFileWasModified(t *testing.T, newObjects []minioClient.ObjectInfo, oldFilename string, oldDate time.Time) {

	for _, obj := range newObjects {
		if obj.Key == oldFilename {
			require.Greater(t, obj.LastModified, oldDate)
			return
		}
	}
	require.Fail(t, "filename not found")
}

func GetGleanerBucketObjects(mc *minioClient.Client, subDir string) ([]minioClient.ObjectInfo, []*minioClient.Object, error) {
	var metadata []minioClient.ObjectInfo
	var objects []*minioClient.Object
	objectCh := mc.ListObjects(context.Background(), "gleanerbucket", minioClient.ListObjectsOptions{Recursive: true, Prefix: subDir})

	for object := range objectCh {
		metadata = append(metadata, object)
		obj, err := mc.GetObject(context.Background(), "gleanerbucket", object.Key, minioClient.GetObjectOptions{})
		if err != nil {
			return nil, nil, err
		}
		objects = append(objects, obj)
	}

	return metadata, objects, nil
}

// Wrapper class over the testcontainer for a cleaner API
type MinioHelper struct {
	Container minio.MinioContainer
	Client    *minioClient.Client
	ctx       context.Context
}

func getAPIURL(container *minio.MinioContainer, ctx context.Context) (string, error) {
	host, err := container.Host(ctx)
	if err != nil {
		return "", err
	}

	api, err := container.MappedPort(ctx, "9000/tcp")
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%s", host, api.Port()), nil
}

func getUIURL(container *minio.MinioContainer, ctx context.Context) (string, error) {
	host, err := container.Host(ctx)
	if err != nil {
		return "", err
	}
	ui, err := container.MappedPort(ctx, "9001/tcp")
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%s", host, ui.Port()), nil
}

// Return the connection strings for the minio API and the minio UI, in that order
func (m *MinioHelper) ConnectionStrings() (string, string, error) {

	apiString, err := getAPIURL(&m.Container, m.ctx)
	if err != nil {
		return "", "", err
	}

	uiString, err := getUIURL(&m.Container, m.ctx)
	if err != nil {
		return "", "", err
	}
	// write this to disk so the user can see it during the test even if verbose logging is off
	uiFile, _ := os.Create("ui_port.txt")
	_, _ = uiFile.WriteString(uiString)
	uiFile.Close()

	return apiString, uiString, nil
}

// Create a handle to a struct which allows for easy handling of the minio container
func NewMinioHandle(img string, opts ...testcontainers.ContainerCustomizer) (*MinioHelper, error) {
	ctx := context.Background()
	const (
		defaultUser     = "minioadmin"
		defaultPassword = "minioadmin"
	)
	req := testcontainers.ContainerRequest{
		Image: img,
		// expose the UI with 9001
		ExposedPorts: []string{"9000/tcp", "9001/tcp"},
		WaitingFor:   wait.ForHTTP("/minio/health/live").WithPort("9000"),
		Env: map[string]string{
			"MINIO_ROOT_USER":     defaultUser,
			"MINIO_ROOT_PASSWORD": defaultPassword,
		},
		// We need to expose the console at 9001 to access the UI
		Cmd: []string{"server", "/data", "--console-address", ":9001"},
	}

	genericContainerReq := testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	}

	for _, opt := range opts {
		if err := opt.Customize(&genericContainerReq); err != nil {
			return nil, err
		}
	}

	username := req.Env["MINIO_ROOT_USER"]
	password := req.Env["MINIO_ROOT_PASSWORD"]
	if username == "" || password == "" {
		return nil, fmt.Errorf("username or password has not been set")
	}

	genericContainer, err := testcontainers.GenericContainer(ctx, genericContainerReq)
	if err != nil {
		return &MinioHelper{}, fmt.Errorf("generic container: %w", err)
	}

	minioContainer := minio.MinioContainer{Container: genericContainer, Username: username, Password: password}

	url, err := getAPIURL(&minioContainer, ctx)

	if err != nil {
		return nil, fmt.Errorf("get api url: %w", err)
	}

	mc, err := minioClient.New(url, &minioClient.Options{
		Creds:  credentials.NewStaticV4(username, password, ""),
		Secure: false,
	})
	if err != nil {
		return nil, fmt.Errorf("minio client: %w", err)
	}

	return &MinioHelper{
		Container: minioContainer,
		Client:    mc,
		ctx:       ctx,
	}, nil
}

func CreateTempGleanerConfig() (string, error) {
	// create a temp config file
	f, err := os.CreateTemp("", "gleanerconfig")
	if err != nil {
		return "", err
	}

	_, err = f.WriteString("gleanerconfig")
	return f.Name(), err
}
