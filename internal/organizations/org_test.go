package organizations

import (
	"testing"

	"gleaner/internal/common"
	config "gleaner/internal/config"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// makes a graph from the Gleaner config file source
// load this to a /sources bucket (change this to sources naming convention?)
func TestBuildGraphMem(t *testing.T) {

	// read config file
	v1, err := config.ReadGleanerConfig("gleanerconfig.yaml", "../../test_helpers")
	assert.NoError(t, err)

	assert.NoError(t, err)
	bucketName, err := config.GetBucketName(v1)
	assert.NoError(t, err)
	assert.Equal(t, "gleanerbucket", bucketName)

	log.Info("Building organization graph from config file")

	domains, err := config.GetSources(v1)
	assert.NoError(t, err)

	assert.Greater(t, len(domains), 0)

	_, _, err = common.GenerateJSONLDProcessor(v1)
	assert.NoError(t, err)

	// for k := range domains {
	// 	// create new S3 file writer
	// 	fw, err := mem.NewMemFileWriter("org.parquet", func(name string, r io.Reader) error {
	// 		dat, err := ioutil.ReadAll(r)
	// 		if err != nil {
	// 			log.Error("error reading data", err)
	// 			return err
	// 		}

	// 		br := bytes.NewReader(dat)

	// 		// load to minio
	// 		objectName := fmt.Sprintf("orgs/%s.parquet", domains[k].Name) // k is the name of the provider from config
	// 		// contentType := "application/ld+json"

	// 		// Upload the file with FPutObject
	// 		_, err = mc.PutObject(context.Background(), bucketName, objectName, br, int64(br.Len()), minio.PutObjectOptions{})
	// 		if err != nil {
	// 			log.Fatal(objectName, err)
	// 			// Fatal?   seriously?  I guess this is the object write, so the run is likely a bust at this point, but this seems a bit much still.
	// 		}

	// 		return err
	// 	})
	// 	if err != nil {
	// 		log.Error("Can't create s3 file writer", err)
	// 		return err
	// 	}

	// 	pw, err := writer.NewParquetWriter(fw, new(Qset), 4)
	// 	assert.NoError(t, err)

	// 	pw.RowGroupSize = 128 * 1024 * 1024 //128M
	// 	pw.PageSize = 8 * 1024              //8K
	// 	// pw.CompressionType = parquet.CompressionCodec_SNAPPY

	// 	// Sources: Name, Logo, URL, Headless, Pid

	// 	jld, err := buildOrgJSONLD(domains[k])
	// 	assert.NoError(t, err)

	// 	r, err := common.JLD2nq(jld, proc, options)
	// 	assert.NoError(t, err)

	// 	// read rdf string line by line and feed into quad decoder

	// 	scanner := bufio.NewScanner(strings.NewReader(r))
	// 	for scanner.Scan() {
	// 		rdfb := bytes.NewBufferString(scanner.Text())
	// 		dec := rdf.NewQuadDecoder(rdfb, rdf.NQuads)

	// 		spog, err := dec.Decode()
	// 		assert.NoError(t, err)

	// 		qs := Qset{Subject: spog.Subj.String(), Predicate: spog.Pred.String(), Object: spog.Obj.String(), Graph: spog.Ctx.String()}

	// 		log.Trace(qs)

	// 		if err = pw.Write(qs); err != nil {
	// 			log.Error("Write error", err)
	// 			return err
	// 		}

	// 	}
	// 	if err := scanner.Err(); err != nil {
	// 		log.Error(err)
	// 		return err
	// 	}

	// 	pw.Flush(true)

	// 	if err = pw.WriteStop(); err != nil {
	// 		log.Error("WriteStop error", err)
	// 		return err
	// 	}

	// 	err = fw.Close()
	// 	if err != nil {
	// 		log.Error("Error closing S3 file writer", err)
	// 		return err
	// 	}

	// 	// delete, is this needed since we close above and have a closure call?
	// 	if err := mem.GetMemFileFs().Remove("org.parquet"); err != nil {
	// 		log.Error("error removing file from memfs:", err)

	// 	}
	// }
}
