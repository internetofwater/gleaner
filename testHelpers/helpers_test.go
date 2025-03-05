package testHelpers

import (
	"context"
	"gleaner/internal/minioWrapper"
	"gleaner/internal/projectpath"
	"path/filepath"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssertLinesMatchDisregardingOrder(
	t *testing.T,
) {

	expected := "hello\nworld\n"
	actual := "world\nhello\n"
	res := AssertLinesMatchDisregardingOrder(expected, actual)
	assert.True(t, res)

	expected = "123456789\n123456789\n"
	actual = "hello\nworld\n"
	res = AssertLinesMatchDisregardingOrder(expected, actual)
	assert.False(t, res)

	expected = "123456789\n123456789\n123"
	actual = "123456789\n123456789\n"
	res = AssertLinesMatchDisregardingOrder(expected, actual)
	assert.False(t, res)
}

func TestDeleteObjects(t *testing.T) {
	minioHandle, err := NewMinioHandle("minio/minio:latest")
	require.NoError(t, err)

	testFile := filepath.Join(projectpath.Root, "testHelpers", "sampleConfigs", "justMainstems.yml")

	// create the gleanerbucket bucket
	client := minioWrapper.MinioClientWrapper{Client: minioHandle.Client, DefaultBucket: "gleanerbucket"}
	err = client.SetupBucket()
	require.NoError(t, err)

	// upload a file
	minioClient := minioHandle.Client
	_, err = minioClient.FPutObject(
		context.Background(),
		"gleanerbucket",
		testFile,
		testFile,
		minio.PutObjectOptions{ContentType: "text/plain"},
	)
	require.NoError(t, err)

	// make sure it exists
	obj, err := minioClient.StatObject(context.Background(), "gleanerbucket", testFile, minio.StatObjectOptions{})
	require.NoError(t, err)

	// delete it
	objects := []minio.ObjectInfo{obj}
	err = DeleteObjects(minioClient, "gleanerbucket", objects)
	require.NoError(t, err)

	// make sure it doesn't exist
	_, err = minioClient.StatObject(context.Background(), "gleanerbucket", testFile, minio.StatObjectOptions{})
	require.Error(t, err)

}
