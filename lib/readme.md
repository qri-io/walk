# crawler lib

```golang
  packge main

  import (
    "github.com/qri-io/crawl/lib"
  )


  func main() {
   flag.Parse()

   crawl := NewCrawl(JSONConfigFromFilepath(cfgPath))

   stop := make(chan bool)
   go stopOnSigKill(stop)

   if err := crawl.Start(stop); err != nil {
     log.Errorf("crawl failed: %s", err.Error())
   }

   if err := crawl.WriteJSON(crawl.cfg.DestPath); err != nil {
     log.Errorf("error writing file: %S", err.Error())
   }

   log.Infof("crawl took: %f hours. wrote %d urls", time.Since(crawl.start).Hours(), crawl.urlsWritten)
  }
```