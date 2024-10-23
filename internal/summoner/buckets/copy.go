package buckets

import (
	"context"
	"fmt"

	"gleaner/internal/common"

	"github.com/spf13/viper"

	minio "github.com/minio/minio-go/v7"
)

// take the bucket name
// look for bucket.1
// if bucket.1 (empty it)
// copy bucket to bucket.1 now
// empty bucket

func list(v1 *viper.Viper) {
	mc := common.MinioConnection(v1)
	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	objectCh := mc.ListObjects(ctx, "mybucket", minio.ListObjectsOptions{
		Prefix:    "myprefix",
		Recursive: true,
	})
	for object := range objectCh {
		if object.Err != nil {
			fmt.Println(object.Err)
			return
		}
		fmt.Println(object)
	}

}
