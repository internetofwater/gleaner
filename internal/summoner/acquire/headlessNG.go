package acquire

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"earthcube.org/Project418/gleaner/internal/common"
	"earthcube.org/Project418/gleaner/pkg/summoner/sitemaps"
	"github.com/mafredri/cdp"
	"github.com/mafredri/cdp/devtool"
	"github.com/mafredri/cdp/protocol/page"
	"github.com/mafredri/cdp/protocol/runtime"
	"github.com/mafredri/cdp/rpcc"
	minio "github.com/minio/minio-go"
	"github.com/spf13/viper"
)

// DocumentInfo contains information about the document.
type DocumentInfo struct {
	Title string `json:"title"`
}

// HeadlessNG gets schema.org entries in sites that put the JSON-LD in dynamically with JS.
// It uses a chrome headless instance (which MUST BE RUNNING).
// TODO..  trap out error where headless is NOT running
func HeadlessNG(v1 *viper.Viper, minioClient *minio.Client, m map[string]sitemaps.Sitemap) {

	for k := range m {
		log.Printf("Headless chrome call to: %s", k)

		for i := range m[k].URL {
			err := run(v1, minioClient, 25*time.Second, m[k].URL[i].Loc, k)
			if err != nil {
				log.Print(err)
			}
		}

	}
}

func run(v1 *viper.Viper, minioClient *minio.Client, timeout time.Duration, url, k string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Use the DevTools HTTP/JSON API to manage targets (e.g. pages, webworkers).
	devt := devtool.New("http://127.0.0.1:9222")
	pt, err := devt.Get(ctx, devtool.Page)
	if err != nil {
		pt, err = devt.Create(ctx)
		if err != nil {
			return err
		}
	}

	// Initiate a new RPC connection to the Chrome DevTools Protocol target.
	conn, err := rpcc.DialContext(ctx, pt.WebSocketDebuggerURL)
	if err != nil {
		return err
	}
	defer conn.Close() // Leaving connections open will leak memory.

	c := cdp.NewClient(conn)

	// Open a DOMContentEventFired client to buffer this event.
	domContent, err := c.Page.DOMContentEventFired(ctx)
	if err != nil {
		return err
	}
	defer domContent.Close()

	// Enable events on the Page domain, it's often preferrable to create
	// event clients before enabling events so that we don't miss any.
	if err = c.Page.Enable(ctx); err != nil {
		return err
	}

	// Create the Navigate arguments with the optional Referrer field set.
	navArgs := page.NewNavigateArgs(url)
	nav, err := c.Page.Navigate(ctx, navArgs)
	if err != nil {
		return err
	}

	// Wait until we have a DOMContentEventFired event.
	if _, err = domContent.Recv(); err != nil {
		return err
	}

	fmt.Printf("Page loaded with frame ID: %s\n", nav.FrameID)

	// Parse information from the document by evaluating JavaScript.
	// querySelector('script[type="application/ld+json"]')
	expression := `
		new Promise((resolve, reject) => {
			setTimeout(() => {
				const title = document.querySelector('#jsonld').innerText;
				resolve({title});
			}, 500);
		});
	`
	evalArgs := runtime.NewEvaluateArgs(expression).SetAwaitPromise(true).SetReturnByValue(true)
	eval, err := c.Runtime.Evaluate(ctx, evalArgs)
	if err != nil {
		fmt.Println(err)
		return (err)
	}

	var info DocumentInfo
	if err = json.Unmarshal(eval.Result.Value, &info); err != nil {
		fmt.Println(err)
		return (err)
	}

	jsonld := info.Title
	fmt.Printf("%s JSON-LD: %s\n\n", url, jsonld)

	if info.Title != "" { // traps out the root domain...   should do this different
		// get sha1 of the JSONLD..  it's a nice ID
		h := sha1.New()
		h.Write([]byte(jsonld))
		bs := h.Sum(nil)
		bss := fmt.Sprintf("%x", bs) // better way to convert bs hex string to string?

		// objectName := fmt.Sprintf("%s/%s.jsonld", up.Path, bss)
		// objectName := fmt.Sprintf("%s.jsonld", bss)
		// objectName := fmt.Sprintf("%s/%s.jsonld", k, bss)
		sha, err := common.GetNormSHA(jsonld, v1) // Moved to the normalized sha value

		objectName := fmt.Sprintf("summoned/%s/%s.jsonld", k, sha)

		contentType := "application/ld+json"
		b := bytes.NewBufferString(jsonld)

		usermeta := make(map[string]string) // what do I want to know?
		usermeta["url"] = url
		usermeta["sha1"] = bss
		bucketName := "gleaner"
		//bucketName := fmt.Sprintf("gleaner-summoned/%s", k) // old was just k

		// Upload the  file with FPutObject
		n, err := minioClient.PutObject(bucketName, objectName, b, int64(b.Len()), minio.PutObjectOptions{ContentType: contentType, UserMetadata: usermeta})
		if err != nil {
			log.Printf("%s", objectName)
			log.Println(err)
		}
		log.Printf("Uploaded Bucket:%s File:%s Size %d \n", bucketName, objectName, n)
	}

	// // Fetch the document root node. We can pass nil here
	// // since this method only takes optional arguments.
	// doc, err := c.DOM.GetDocument(ctx, nil)
	// if err != nil {
	// 	return err
	// }

	// // Get the outer HTML for the page.
	// // #jsonld
	// // document.querySelector("#jsonld")
	// // //*[@id="jsonld"]
	// // /html/head/script[5]
	// result, err := c.DOM.GetOuterHTML(ctx, &dom.GetOuterHTMLArgs{
	// 	NodeID: &doc.Root.NodeID,
	// })
	// if err != nil {
	// 	return err
	// }

	// fmt.Printf("HTML: %s\n", len(result.OuterHTML))

	return nil
}
