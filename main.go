package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
)

type DomainRecord struct {
	Domain    string `json:"domain"`
	Backlinks string `json:"backlinks"`
	DomainPop string `json:"domain_pop"`
	DropDate  string `json:"drop_date"`
}

type OutputData struct {
	Metadata struct {
		TotalDomains int `json:"total_domains"`
	} `json:"metadata"`
	Domains []DomainRecord `json:"domains"`
}

func runScraper() {
	var records []DomainRecord
	var mu sync.Mutex

	c := colly.NewCollector(
		colly.AllowedDomains("www.expireddomains.net", "expireddomains.net"),
		colly.Async(true),
	)

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*expireddomains.net*",
		Parallelism: 4,
		RandomDelay: 2,
	})

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
		fmt.Println("now visiting:", r.URL)
	})

	c.OnHTML("table.datatable tbody tr, table.tldtable tbody tr", func(e *colly.HTMLElement) {
		domain := e.ChildText("td.l a")
		backlinks := e.ChildText("td.itz")
		domainPop := e.ChildText("td.ip")
		dropdate := e.ChildText("td.bgd")

		if domain != "" {
			mu.Lock()
			records = append(records, DomainRecord{
				Domain:    domain,
				Backlinks: backlinks,
				DomainPop: domainPop,
				DropDate:  dropdate,
			})
			mu.Unlock()
			fmt.Printf("new domain found: %s | date: %s\n", domain, dropdate)
		}
	})

	c.OnHTML("a.next", func(e *colly.HTMLElement) {
		nxtpage := e.Attr("href")
		if nxtpage != "" {
			e.Request.Visit(nxtpage)
		}
	})

	c.Visit("https://www.expireddomains.net/tld/com/")

	c.Wait()

	output := OutputData{
		Domains: records,
	}
	output.Metadata.TotalDomains = len(records)

	fileData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Println("JSON marshaling failed:", err)
		return
	}
	
	err = os.WriteFile("domains.json", fileData, 0644)
	if err != nil {
		fmt.Println("Failed to write file:", err)
		return
	}
	
	fmt.Printf("Successfully saved %d records to domains.json\n", len(records))
}

func main() {
	fmt.Println("Starting async domain scraper daemon...")
	
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		runScraper()
		fmt.Println("Sleeping until next cycle...")
		<-ticker.C
	}
}
