package main

import (
  "context"
  "html/template"
  "log"
  "net/http"
  "os"

  "cloud.google.com/go/firestore"
)

type Domains struct {
  Tld     string
  Domains []*Domain
}
type Domain struct {
  Domain  string
  Details map[string]any
}

var projectid = os.Getenv("GOOGLE_CLOUD_PROJECT")

func toppage(w http.ResponseWriter, r *http.Request) {
  ctx := context.Background()

  log.Printf("Initialize Firestore client for project: %s", projectid)
  client, err := firestore.NewClient(ctx, projectid)
  if err != nil {
    log.Printf("Firestore client initialization failed: %s", err)
    http.Error(w, "Internal Server Error", http.StatusInternalServerError)
    return
  }

  defer client.Close()

  var domains []Domains
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

    domains = append(domains, Domains{tld, records})
  }

  w.Header().Set("Cache-Control", "public, max-age=600") // Ask Goole Frontend to cache the page

  err = template.Must(template.ParseFiles("template.html")).Execute(w, domains)
  if err != nil {
    log.Printf("Template execution error: %s", err)
    http.Error(w, "Internal Server Error", http.StatusInternalServerError)
    return
  }
}

func main() {
  http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case "GET":
      if r.URL.Path != "/" {
        http.Error(w, "Not Found", http.StatusNotFound)
        return
      }
      toppage(w, r)

    default:
      http.Error(w, "Not Implemented", http.StatusNotImplemented)
    }
  })

  port := os.Getenv("PORT")
  if port == "" {
    port = "8080"
    log.Printf("Defaulting to port %s", port)
  }

  log.Printf("Listening on port %s", port)
  err := http.ListenAndServe(":"+port, nil)
  if err != nil {
    log.Fatal(err)
  }
}
