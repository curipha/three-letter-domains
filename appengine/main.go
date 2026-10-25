package main

import (
  "bytes"
  "context"
  "html/template"
  "log"
  "net/http"
  "os"
  "time"

  "cloud.google.com/go/firestore"
)

type TLDGroup struct {
  Tld     string
  Domains []*Domain
}
type Domain struct {
  Domain  string
  Details map[string]any
}

var (
  client *firestore.Client
  projectid = os.Getenv("GOOGLE_CLOUD_PROJECT")
  tmpl = template.Must(template.ParseFiles("template.html"))
)

func toppage(w http.ResponseWriter, r *http.Request) {
  ctx := r.Context()

  var tldgrps []TLDGroup
  for _, tld := range []string{"com", "net"} {
    query := client.Collection(tld).OrderBy("expiration", firestore.Asc).Limit(20)
    docs, err := query.Documents(ctx).GetAll()
    if err != nil {
      log.Printf("Failed to query to Firestore: %s", err)
      http.Error(w, "Internal Server Error", http.StatusInternalServerError)
      return
    }

    var records []*Domain
    for _, doc := range docs {
      records = append(
        records,
        &Domain{
          Domain:  doc.Ref.ID,
          Details: doc.Data(),
        },
      )
    }

    tldgrps = append(tldgrps, TLDGroup{tld, records})
  }


  var buf bytes.Buffer
  err := tmpl.Execute(&buf, tldgrps)
  if err != nil {
    log.Printf("Template execution error: %v", err)
    http.Error(w, "Internal Server Error", http.StatusInternalServerError)
    return
  }

  w.Header().Set("Cache-Control", "public, max-age=600") // Ask Google Frontend to cache the page
  buf.WriteTo(w)
}

func main() {
  var err error
  log.Printf("Initialize Firestore client for project: %s", projectid)
  client, err = firestore.NewClient(context.Background(), projectid)
  if err != nil {
    log.Fatalf("Firestore client initialization failed: %v", err)
    return
  }

  defer client.Close()


  mux := http.NewServeMux()

  // "GET" also accepts HEAD; other methods return 405 and an Allow header.
  // "{$}" makes paths other than "/" return a 404.
  mux.HandleFunc("GET /{$}", toppage)

  port := os.Getenv("PORT")
  if port == "" {
    port = "8080"
    log.Printf("Defaulting to port %s", port)
  }


  server := &http.Server{
    Addr:              ":" + port,
    Handler:           mux,
    ReadHeaderTimeout: 10 * time.Second, // Mitigate Slowloris
    ReadTimeout:       10 * time.Second,
    WriteTimeout:      10 * time.Second,
    IdleTimeout:       60 * time.Second,
  }

  log.Printf("Listening on port %s", port)
  err = server.ListenAndServe()
  if err != nil {
    log.Fatal(err)
  }
}
