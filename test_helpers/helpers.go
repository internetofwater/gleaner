package test_helpers

import (
	"context"
	"fmt"
	"testing"

	minioClient "github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/minio"
	"github.com/testcontainers/testcontainers-go/wait"
)

func AssertObjectCount(t *testing.T, mc *minioClient.Client, subDir string, expected int) {

	_, summoned, err := GetGleanerBucketObjects(mc, subDir)
	assert.NoError(t, err)
	assert.Equal(t, expected, len(summoned))

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

func ConnectionStrings(ctx context.Context, c *minio.MinioContainer) (string, string, error) {
	host, err := c.Host(ctx)
	if err != nil {
		return "", "", err
	}
	port, err := c.MappedPort(ctx, "9000/tcp")
	if err != nil {
		return "", "", err
	}
	ui, err := c.MappedPort(ctx, "9001/tcp")
	if err != nil {
		return "", "", err
	}
	return fmt.Sprintf("%s:%s", host, port.Port()), fmt.Sprintf("%s:%s", host, ui.Port()), nil
}

// Run creates an instance of the Minio container type
func MinioRun(ctx context.Context, img string, opts ...testcontainers.ContainerCustomizer) (*minio.MinioContainer, error) {
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

	container, err := testcontainers.GenericContainer(ctx, genericContainerReq)
	var c *minio.MinioContainer
	if container != nil {
		c = &minio.MinioContainer{Container: container, Username: username, Password: password}
	}

	if err != nil {
		return c, fmt.Errorf("generic container: %w", err)
	}

	return c, nil
}
