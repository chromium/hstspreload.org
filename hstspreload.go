package hstspreload

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"appengine"
	"appengine/datastore"
	"appengine/urlfetch"
	"appengine/user"

	"golang.org/x/net/publicsuffix"
)

func init() {
	http.HandleFunc("/", index)
	http.HandleFunc("/style.css", styleCSS)
	http.HandleFunc("/submit/", submit)
	http.HandleFunc("/clear/", clear)
	http.HandleFunc("/pending", pending)
	http.HandleFunc("/update", update)
	http.HandleFunc("/setmessage", setMessage)
	http.HandleFunc("/setmessages", setMessages)
	http.HandleFunc("/robots.txt", gen404)
	http.HandleFunc("/favicon.ico", gen404)
}

func gen404(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "No such resource", 404)
}

func serveFile(w http.ResponseWriter, r *http.Request, fileName string, contentType string) {
	page, err := os.Open(fileName)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer page.Close()

	w.Header().Set("Content-type", contentType)
	io.Copy(w, page)
}

func index(w http.ResponseWriter, r *http.Request) {
	serveFile(w, r, "page.html", "text/html; charset=utf-8")
}

func styleCSS(w http.ResponseWriter, r *http.Request) {
	serveFile(w, r, "style.css", "text/css; charset=utf-8")
}

type replyJSON struct {
	Canon        string `json:",omitempty"`
	err          error  `json:"-"`
	Error        string `json:",omitempty"`
	IsPending    bool   `json:",omitempty"`
	IsPreloaded  bool   `json:",omitempty"`
	Exception    string `json:",omitempty"`
	NoHeader     bool   `json:",omitempty"`
	WasRedirect  bool   `json:",omitempty"`
	NoPreload    bool   `json:",omitempty"`
	NoSubdomains bool   `json:",omitempty"`
	MaxAge       uint64 `json:",omitempty"`
	Accepted     bool   `json:",omitempty"`
}

const (
	statusUnknown = iota
	statusPending
	statusPreloaded
	statusException
)

type HostState struct {
	Status  int
	Message string
}

func submit(w http.ResponseWriter, r *http.Request) {
	domain := r.URL.Path[8:]

	reply := handleDomain(domain, r)
	if reply.err != nil {
		reply.Error = reply.err.Error()
	}

	w.Header().Set("Content-type", "application/json")

	encoder := json.NewEncoder(w)
	if err := encoder.Encode(reply); err != nil {
		panic(err)
	}
}

func clear(w http.ResponseWriter, r *http.Request) {
	domain := r.URL.Path[7:]

	if !handleClear(domain, r) {
		http.Error(w, "Internal error", 409)
		return
	}
}

func pending(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	w.Header().Set("Content-type", "text/plain; charset=utf-8")

	query := datastore.NewQuery("HostState").Filter("Status =", statusPending).KeysOnly()
	iter := query.Run(c)

	for {
		key, err := iter.Next(nil)
		if err == datastore.Done {
			break
		}
		if err != nil {
			fmt.Fprintf(w, "Error from iterator: %s\n", err.Error())
			break
		}
		fmt.Fprintf(w, `    { "name": "%s", "include_subdomains": true, "mode": "force-https" },`, key.StringID())
		w.Write(newLine)
	}
}

type preloaded struct {
	Entries []hsts `json:"entries"`
}

type hsts struct {
	Name       string `json:"name"`
	Subdomains bool   `json:"include_subdomains"`
	Mode       string `json:"mode"`
	Pins       string `json:"pins"`
	SNIOnly    bool   `json:"snionly"`
}

func newClient(context appengine.Context, t time.Duration) *http.Client {
	return &http.Client{
		Transport: &urlfetch.Transport{
			Context:  context,
			Deadline: t,
		},
	}
}

func update(w http.ResponseWriter, r *http.Request) {
	const url = "https://chromium.googlesource.com/chromium/src/+/master/net/http/transport_security_state_static.json?format=TEXT"

	c := appengine.NewContext(r)
	client := newClient(c, time.Second*15)
	resp, err := client.Get(url)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if resp.StatusCode != 200 {
		http.Error(w, fmt.Sprintf("Status code %d", resp.StatusCode), 500)
		return
	}

	jsonBytes, err := removeComments(base64.NewDecoder(base64.StdEncoding, resp.Body))
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	var preloaded preloaded
	if err := json.Unmarshal(jsonBytes, &preloaded); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	var keys []*datastore.Key
	var values []HostState

	for _, entry := range preloaded.Entries {
		if entry.SNIOnly || entry.Mode != "force-https" {
			continue
		}
		keys = append(keys, datastore.NewKey(c, "HostState", entry.Name, 0, nil))
		values = append(values, HostState{Status: statusPreloaded})
		if len(keys) > 450 {
			if _, err := datastore.PutMulti(c, keys, values); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			keys = keys[:0]
			values = values[:0]
		}
	}

	if len(keys) > 0 {
		if _, err := datastore.PutMulti(c, keys, values); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	}

	w.Header().Set("Content-type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "done.\n")
}

// commentRegexp matches lines that optionally start with whitespace
// followed by "//".
var commentRegexp = regexp.MustCompile("^[ \t]*//")

var newLine = []byte("\n")

// removeComments reads the contents of |r| and removes any lines beginning
// with optional whitespace followed by "//"
func removeComments(r io.Reader) ([]byte, error) {
	var buf bytes.Buffer
	in := bufio.NewReader(r)

	for {
		line, isPrefix, err := in.ReadLine()
		if isPrefix {
			return nil, errors.New("line too long in JSON")
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if commentRegexp.Match(line) {
			continue
		}
		buf.Write(line)
		buf.Write(newLine)
	}

	return buf.Bytes(), nil
}

type redirectStopError struct{}

func (redirectStopError) Error() string {
	return "stopped redirect"
}

var redirectStop = redirectStopError{}

func checkRedirect(req *http.Request, via []*http.Request) error {
	return redirectStop
}

func handleDomain(domain string, r *http.Request) replyJSON {
	if strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") || strings.Index(domain, "..") != -1 {
		return replyJSON{err: errors.New("domain ill formed")}
	}

	if strings.Count(domain, ".") < 1 {
		return replyJSON{err: errors.New("domain must have at least two labels")}
	}

	domain = strings.ToLower(domain)
	for _, r := range domain {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '.' {
			continue
		}
		return replyJSON{err: errors.New("domain contains invalid characters")}
	}

	canon, err := publicsuffix.EffectiveTLDPlusOne(domain)
	if err != nil {
		return replyJSON{err: err}
	}
	if canon != domain {
		return replyJSON{Canon: canon}
	}

	c := appengine.NewContext(r)
	key := datastore.NewKey(c, "HostState", domain, 0, nil)

	var state HostState
	if err := datastore.Get(c, key, &state); err != nil && err != datastore.ErrNoSuchEntity {
		return replyJSON{err: err}
	}

	switch state.Status {
	case statusPending:
		return replyJSON{IsPending: true}
	case statusPreloaded:
		return replyJSON{IsPreloaded: true}
	case statusException:
		return replyJSON{Exception: state.Message}
	}

	client := urlfetch.Client(c)
	client.CheckRedirect = checkRedirect
	resp, err := client.Get("https://" + domain)
	wasRedirect := false
	if err != nil {
		if urlError, ok := err.(*url.Error); ok && urlError.Err == redirectStop {
			wasRedirect = true
		} else {
			fmt.Printf("%T %#v\n", err)
			return replyJSON{err: err}
		}
	}

	if r := resp.StatusCode; r >= 400 && r < 599 {
		return replyJSON{err: fmt.Errorf("fetch returned status %s", resp.Status)}
	}

	hsts := resp.Header.Get("Strict-Transport-Security")
	if len(hsts) == 0 {
		return replyJSON{NoHeader: true, WasRedirect: wasRedirect}
	}

	hstsParts := strings.Split(hsts, ";")
	for i, part := range hstsParts {
		hstsParts[i] = strings.TrimSpace(part)
	}

	hasPreload := false
	hasIncludeSubdomains := false
	maxAge := uint64(0)
	for _, part := range hstsParts {
		part = strings.ToLower(part)
		if part == "preload" {
			hasPreload = true
		} else if part == "includesubdomains" {
			hasIncludeSubdomains = true
		} else if strings.HasPrefix(part, "max-age=") {
			maxAge, err = strconv.ParseUint(part[8:], 10, 64)
			if err != nil {
				return replyJSON{err: errors.New("failed to parse max-age value: " + err.Error())}
			}
		}
	}

	if maxAge < 10886400 || !hasPreload || !hasIncludeSubdomains {
		return replyJSON{
			MaxAge:       maxAge,
			NoPreload:    !hasPreload,
			NoSubdomains: !hasIncludeSubdomains,
		}
	}

	state.Status = statusPending
	if _, err := datastore.Put(c, key, &state); err != nil {
		return replyJSON{err: err}
	}

	return replyJSON{Accepted: true}
}

func handleClear(domain string, r *http.Request) bool {
	c := appengine.NewContext(r)
	key := datastore.NewKey(c, "HostState", domain, 0, nil)

	err := datastore.RunInTransaction(c, func(c appengine.Context) error {
		var state HostState
		if err := datastore.Get(c, key, &state); err != nil {
			return err
		}
		if state.Status != statusException {
			return errors.New("incorrect state")
		}
		return datastore.Delete(c, key)
	}, nil)

	if err != nil {
		fmt.Printf("clear transaction for %q resulted in error: %s", domain, err)
		return false
	}

	return true
}

func setMessage(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	u := user.Current(c)
	if u == nil {
		url, _ := user.LoginURL(c, "/setmessage")
		http.Redirect(w, r, url, 302)
		return
	}

	if !strings.HasSuffix(u.String(), "@google.com") {
		http.Error(w, "Not authorised", 403)
		return
	}

	domain := r.FormValue("domain")
	msg := r.FormValue("msg")

	if len(domain) == 0 || len(msg) == 0 {
		page, err := os.Open("msg.html")
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer page.Close()

		w.Header().Set("Content-type", "text/html; charset=utf-8")
		io.Copy(w, page)
		return
	}

	if (r.Header.Get("referer") != "https://hstspreload.appspot.com/setmessage") {
		http.Error(w, "Not authorised", 403)
		return
	}

	key := datastore.NewKey(c, "HostState", domain, 0, nil)
	state := HostState{
		Status:  statusException,
		Message: msg,
	}

	if _, err := datastore.Put(c, key, &state); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-type", "text/plain; charset=utf-8")
	w.Write([]byte("done."))
}

func setMessages(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	u := user.Current(c)
	if u == nil {
		url, _ := user.LoginURL(c, "/setmessages")
		http.Redirect(w, r, url, 302)
		return
	}

	if !strings.HasSuffix(u.String(), "@google.com") {
		http.Error(w, "Not authorised", 403)
		return
	}

	jsonData := []byte(r.FormValue("json"))

	if len(jsonData) == 0 {
		page, err := os.Open("msgs.html")
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer page.Close()

		w.Header().Set("Content-type", "text/html; charset=utf-8")
		io.Copy(w, page)
		return
	}

	if (r.Header.Get("referer") != "https://hstspreload.appspot.com/setmessages") {
		http.Error(w, "Not authorised", 403)
		return
	}

	type SetMessageJSON struct {
		Name    string
		Message string
	}

	var messages []SetMessageJSON
	if err := json.Unmarshal(jsonData, &messages); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	var keys []*datastore.Key
	var values []HostState

	for _, msg := range messages {
		keys = append(keys, datastore.NewKey(c, "HostState", msg.Name, 0, nil))
		values = append(values, HostState{Status: statusException, Message: msg.Message})
		if len(keys) > 450 {
			if _, err := datastore.PutMulti(c, keys, values); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			keys = keys[:0]
			values = values[:0]
		}
	}

	if len(keys) > 0 {
		if _, err := datastore.PutMulti(c, keys, values); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	}

	w.Header().Set("Content-type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "done.\n")
}
