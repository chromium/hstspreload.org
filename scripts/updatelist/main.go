package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/chromium/hstspreload"
	"github.com/chromium/hstspreload/chromium/preloadlist"
	"golang.org/x/sync/errgroup"
)

type PendingChanges struct {
	pendingAdditions         []string
	pendingRemovals          []string
	pendingAutomatedRemovals []string
	removals                 map[string]bool
}

func fetchPendingChanges() (*PendingChanges, error) {
	changes := new(PendingChanges)
	g := new(errgroup.Group)
	g.Go(func() error {
		log.Println("Fetching pending additions...")
		resp, err := http.Get("https://hstspreload.org/api/v2/pending")
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		pendingReader := json.NewDecoder(resp.Body)
		pendingEntries := []preloadlist.Entry{}
		if err := pendingReader.Decode(&pendingEntries); err != nil {
			return err
		}
		for _, entry := range pendingEntries {
			changes.pendingAdditions = append(changes.pendingAdditions, entry.Name)
		}
		slices.Sort(changes.pendingAdditions)
		return nil
	})
	g.Go(func() error {
		log.Println("Fetching pending removals...")
		resp, err := http.Get("https://hstspreload.org/api/v2/pending-removal")
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		pendingReader := json.NewDecoder(resp.Body)
		if err := pendingReader.Decode(&changes.pendingRemovals); err != nil {
			return err
		}
		return nil
	})
	g.Go(func() error {
		log.Println("Fetching pending automated removals...")
		resp, err := http.Get("https://hstspreload.org/api/v2/pending-automated-removal")
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		pendingReader := json.NewDecoder(resp.Body)
		if err := pendingReader.Decode(&changes.pendingAutomatedRemovals); err != nil {
			return err
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}
	log.Println("... all fetches complete")
	changes.updateRemovals()
	return changes, nil
}

func (pc *PendingChanges) updateRemovals() {
	pc.removals = make(map[string]bool)
	for _, r := range pc.pendingRemovals {
		pc.removals[r] = true
	}
	for _, r := range pc.pendingAutomatedRemovals {
		pc.removals[r] = true
	}
}

// Filter modifies the list of pending changes to include only domains that
// still meet the criteria for that domain's proposed state.
func (pc *PendingChanges) Filter() {
	log.Print("Verifying pending additions...")
	pc.pendingAdditions = filterParallel(pc.pendingAdditions, func(domain string) bool {
		// A pending addition to the list is still valid to add to the list if scanning the domain indicates no errors.
		_, issues := hstspreload.EligibleDomain(domain, preloadlist.Bulk1Year)
		return len(issues.Errors) == 0
	})
	log.Print("Verifying pending automated removals...")
	pc.pendingAutomatedRemovals = filterParallel(pc.pendingAutomatedRemovals, func(domain string) bool {
		// TODO: the call to EligibleDomain should be made with the policy that
		// the domain was originally preloaded with. The pending-automated-removal
		// endpoint does not expose that policy information. To prevent from
		// incorrectly removing old entries added with the 18-week policy, this
		// check always uses the 18-week policy.
		_, issues := hstspreload.EligibleDomain(domain, preloadlist.Bulk18Weeks)

		// A pending automated removal is eligible for removal if it continues
		// to not meet the preload requirements, i.e. it has errors.
		return len(issues.Errors) > 0
	})
	log.Print("... done verifying domains")
	pc.updateRemovals()
}

type tickLogger struct {
	ticker  *time.Ticker
	logLine string
	done    chan bool
}

func (t *tickLogger) Logf(format string, v ...any) {
	t.logLine = fmt.Sprintf(format, v...)
}

func (t *tickLogger) Stop() {
	t.ticker.Stop()
	t.done <- true
}

func newTickLogger(d time.Duration) *tickLogger {
	t := new(tickLogger)
	t.ticker = time.NewTicker(d)
	t.done = make(chan bool)
	go func() {
		for {
			select {
			case <-t.done:
				return
			case <-t.ticker.C:
				if t.logLine == "" {
					return
				}
				log.Print(t.logLine)
			}
		}
	}()
	return t
}

func filterParallel(domains []string, predicate func(domain string) bool) []string {
	mu := sync.Mutex{}
	filtered := make([]string, 0)

	parallelism := 500
	sem := make(chan any, parallelism) // Use a buffered channel to limit the amount of parallelism
	wg := sync.WaitGroup{}
	l := newTickLogger(5 * time.Second)
	defer l.Stop()
	for i, domain := range domains {
		l.Logf("started processing %d domains", i)
		sem <- nil // Acquire a slot
		wg.Add(1)
		go func(domain string) {
			defer func() {
				wg.Done()
				<-sem // Release the slot
			}()
			if predicate(domain) {
				mu.Lock()
				filtered = append(filtered, domain)
				mu.Unlock()
			}
		}(domain)
	}
	wg.Wait()
	return filtered
}

// PendingAdditions returns a sorted list of domain names that are pending
// addition to the HSTS preload list.
func (pc *PendingChanges) PendingAdditions() []string {
	return pc.pendingAdditions
}

func (pc *PendingChanges) Removes(domain string) bool {
	return pc.removals[domain]
}

type dupeTracker struct {
	seenDomains map[string]int
}

func (d *dupeTracker) Observe(domain string) {
	if d.seenDomains == nil {
		d.seenDomains = make(map[string]int)
	}
	d.seenDomains[domain]++
}

func (d *dupeTracker) Dupes() []string {
	domains := []string{}
	for domain, count := range d.seenDomains {
		if count < 2 {
			continue
		}
		domains = append(domains, domain)
	}
	slices.Sort(domains)
	return domains
}

func updateList(listContents []byte, changes *PendingChanges) (string, []string, error) {
	listString := strings.TrimSuffix(string(listContents), "\n")
	log.Print("Removing and adding entries...")
	commentRe := regexp.MustCompile("^ *//.*")
	listEntryRe := regexp.MustCompile(`^    \{.*\},`)
	output := strings.Builder{}
	dupes := dupeTracker{}
	for _, line := range strings.Split(listString, "\n") {
		if commentRe.MatchString(line) {
			if line != "    // END OF 1-YEAR BULK HSTS ENTRIES" {
				output.WriteString(line)
				output.WriteByte('\n')
				continue
			}
			for _, domain := range changes.PendingAdditions() {
				dupes.Observe(domain)
				fmt.Fprintf(&output, `    { "name": "%s", "policy": "bulk-1-year", "mode": "force-https", "include_subdomains": true },`, domain)
				fmt.Fprintln(&output)
			}
			output.WriteString(line)
			output.WriteByte('\n')
			continue
		}
		if !listEntryRe.MatchString(line) {
			output.WriteString(line)
			output.WriteByte('\n')
			continue
		}
		entry := preloadlist.Entry{}
		if err := json.Unmarshal([]byte(strings.TrimSuffix(line, ",")), &entry); err != nil {
			return "", nil, err
		}
		if !changes.Removes(entry.Name) {
			dupes.Observe(entry.Name)
			output.WriteString(line)
			output.WriteByte('\n')
		}
	}
	return output.String(), dupes.Dupes(), nil
}

func overwriteFile(f *os.File, contents string) error {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if err := f.Truncate(0); err != nil {
		return err
	}
	if _, err := f.WriteString(contents); err != nil {
		return err
	}
	return nil
}

func main() {
	listPath := flag.String("list_path", "", "Path to the file containing the HSTS preload list")
	flag.Parse()
	if *listPath == "" {
		log.Fatal("list_path not specified")
	}

	// Open the JSON file containing the HSTS preload list.
	log.Print("Fetching preload list from Chromium source...")
	listFile, err := os.OpenFile(*listPath, os.O_RDWR, 0)
	if err != nil {
		log.Fatalf("Failed to open HSTS preload list in %q: %v", *listPath, err)
	}
	defer listFile.Close()

	listContents, err := io.ReadAll(listFile)
	if err != nil {
		log.Fatalf("Failed to read HSTS preload list: %v", err)
	}

	// fetch pending changes from hstspreload.org
	changes, err := fetchPendingChanges()
	if err != nil {
		log.Fatalf("Error fetching pending changes from hstspreload.org: %v", err)
	}

	// filter changes to only ones that are still valid
	changes.Filter()

	// apply the changes to the JSON file in the chromium source
	updatedList, dupes, err := updateList(listContents, changes)
	if err != nil {
		log.Fatalf("Failed to update list: %v", err)
	}
	if err := overwriteFile(listFile, updatedList); err != nil {
		log.Fatalf("Error writing HSTS preload list file: %v", err)
	}

	if len(dupes) > 0 {
		fmt.Println("WARNING\nDuplicate entries:")
		for _, dupe := range dupes {
			fmt.Printf("- %s\n", dupe)
		}
		fmt.Println("You'll need to manually deduplicate entries before commiting them to Chromium")
		fmt.Println("Note: if there are a lot of duplicate entries, you may have accidentally run this program twice. Reset your checkout and try again.")
	} else {
		fmt.Println("SUCCESS")
	}
}
