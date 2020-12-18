package main

import (
    "fmt"
    "io"
    "log"
    "os"
    "net/http"
    "strings"

    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
    "github.com/aws/aws-sdk-go/service/s3/s3manager"
)

type FakeWriterAt struct {
    w io.Writer
}

func (fw FakeWriterAt) WriteAt(p []byte, offset int64) (n int, err error) {
    // ignore 'offset' because we forced sequential downloads
    return fw.w.Write(p)
}

func faviconHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "/app/favicon.ico")
}

func listHandler(w http.ResponseWriter, r *http.Request, svc *s3.S3) {

    paths := strings.SplitN(r.URL.Path[1:], "/", 3)
    paths = paths[1:]
    fmt.Println(paths)

    if paths[0] == "" {
        fmt.Fprintf(w, "<html><body>\n")
        fmt.Println(r.URL.Path[1:])
        fmt.Fprintf(w, "<p>Contents of %q</p>\n", strings.SplitN(r.URL.Path[1:], "/", 2))
        fmt.Println("Listing buckets...")
        input := &s3.ListBucketsInput{}

        result, err := svc.ListBuckets(input)
        if err != nil {
            if aerr, ok := err.(awserr.Error); ok {
                switch aerr.Code() {
                default:
                    fmt.Println(aerr.Error())
                }
            } else {
                fmt.Println(err.Error())
            }
            return
        }

        fmt.Fprintf(w, "BUCKETS<br />\n")
        for _, b := range result.Buckets {
            fmt.Fprintf(w, "<a href=\"/browse/%s/\">%s</a>\t\t%s<br />\n", *b.Name, *b.Name, b.CreationDate)
        }
        fmt.Fprintf(w, "</body></html>\n")
    } else if strings.HasSuffix(r.URL.Path, "/") {
        fmt.Fprintf(w, "<html><body>\n")
        fmt.Fprintf(w, "<p>Contents of %q</p>\n", strings.SplitN(r.URL.Path[1:], "/", 2))
        fmt.Println(paths)
        delim := "/"
        pfx := ""
        if len(paths) > 1 {
            pfx = paths[1]
        }

        fmt.Fprintf(w, "<a href=\"..\">../</a><br />\n")
        err := svc.ListObjectsV2Pages(&s3.ListObjectsV2Input{
            Bucket: &paths[0],
            Prefix: &pfx,
            Delimiter: &delim,
        }, func(p *s3.ListObjectsV2Output, _ bool) (shouldContinue bool) {
            for _, c := range p.Contents {
                fmt.Fprintf(w, "<a href=\"%s\">%s</a>\t\t%d\t\t%s<br />\n", *c.Key, *c.Key, *c.Size, c.LastModified)
            }
            for _, c := range p.CommonPrefixes {
                fmt.Fprintf(w, "<a href=\"/browse/%s/%s\">%s</a><br />\n", paths[0], *c.Prefix, *c.Prefix)
            }
            return true
        })
        if err != nil {
            fmt.Println("failed to list objects", err)
            return
        }

        fmt.Fprintf(w, "</body></html>\n")
    } else {
        fmt.Println("GET %s", paths[1])
        downloader := s3manager.NewDownloaderWithClient(svc)
        downloader.Concurrency = 1

        _, err := downloader.Download(FakeWriterAt{w},
            &s3.GetObjectInput{
                Bucket: &paths[0],
                Key:    &paths[1],
            })
        if err != nil {
            fmt.Println("Unable to download item %q, %v", paths, err)
        }

    }
}

func searchHandler(w http.ResponseWriter, r *http.Request, svc *s3.S3) {

    paths := strings.SplitN(r.URL.Path[1:], "/", 3)
    bucket := r.URL.Query()["bucket"][0]
    if bucket == "" {
        bucket = paths[1]
    }
    query := r.URL.Query()["query"][0]
    if query == "" {
        query = strings.Join(paths[2:], "/")
    }
    fmt.Println(query)

    fmt.Fprintf(w, "<html><body>\n")
    fmt.Fprintf(w, "<p>Search results for %q</p>\n", strings.SplitN(r.URL.Path[1:], "/", 2))
    fmt.Fprintf(w, "<a href=\"..\">../</a><br />\n")

    err := svc.ListObjectsV2Pages(&s3.ListObjectsV2Input{
        Bucket: &bucket,
        Prefix: &query,
    }, func(p *s3.ListObjectsV2Output, _ bool) (shouldContinue bool) {
        for _, c := range p.Contents {
            fmt.Fprintf(w, "<a href=\"/browse/%s/%s\">%s</a>\t\t%d\t\t%s<br />\n", bucket, *c.Key, *c.Key, *c.Size, c.LastModified)
        }
        return true
    })
    if err != nil {
        fmt.Println("failed to list objects", err)
        return
    }

    fmt.Fprintf(w, "</body></html>\n")
}

func rootHandler(w http.ResponseWriter, r *http.Request) {

    fmt.Fprintf(w, "<html><body>\n")
    fmt.Fprintf(w, "<form action=\"/search\" method=\"get\">\n")
    fmt.Fprintf(w, "<ul><li><label for=\"bucket\">Bucket:</label><input type=\"text\" id=\"bucket\" name=\"bucket\"></li>")
    fmt.Fprintf(w, "<li><label for=\"query\">Query:</label><input type=\"text\" id=\"query\" name=\"query\"></li></ul>")
    fmt.Fprintf(w, "<li class=\"button\"><button type=\"submit\">Send your message</button></li></ul>")
    fmt.Fprintf(w, "</form>\n")
    fmt.Fprintf(w, "<hr>\n")
    fmt.Fprintf(w, "Or just <a href=\"/browse/\">Browse</a> buckets and objects<br />\n")
    fmt.Fprintf(w, "</body></html>\n")
}

func main() {

    endpoint := os.Getenv("S3ENDPOINT")
    s3Config := &aws.Config{
        Endpoint:         aws.String(endpoint),
        Region:           aws.String("us-east-1"),
        DisableSSL:       aws.Bool(true),
        S3ForcePathStyle: aws.Bool(true),
    }

    sess := session.Must(session.NewSession(s3Config))
    svc := s3.New(sess)

    http.HandleFunc("/", rootHandler)
    http.HandleFunc("/favicon.ico", faviconHandler)
    http.HandleFunc("/browse/", func(w http.ResponseWriter, r *http.Request) { listHandler(w, r, svc) })
    http.HandleFunc("/search/", func(w http.ResponseWriter, r *http.Request) { searchHandler(w, r, svc) })
    log.Fatal(http.ListenAndServe(":8080", nil))
}
