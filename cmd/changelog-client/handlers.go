package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
)

// Known stable release streams to display on the index page.
var defaultStreams = []string{
	"4-stable",
	"4-stable-arm64",
	"4-stable-multi",
	"4-stable-s390x",
	"4-stable-ppc64le",
}

type Server struct {
	client  *Client
	streams []string
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "ok")
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	streams, err := s.client.GetReleaseStreams()
	if err != nil {
		log.Printf("Error fetching release streams, using defaults: %v", err)
		streams = s.streams
	}

	w.Header().Set("Content-Type", "text/html;charset=UTF-8")
	renderIndexPage(w, s.client.rcURL, streams)
}

func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	stream, _, hasRelease := parseReleasePath(r.URL.Path)
	if stream == "" {
		http.NotFound(w, r)
		return
	}

	if hasRelease {
		s.handleRelease(w, r, stream)
		return
	}

	// Stream list page
	tags, err := s.client.GetReleaseTags(stream)
	if err != nil {
		log.Printf("Error fetching tags for %s: %v", stream, err)
		http.Error(w, fmt.Sprintf("Unable to fetch stream %s: %v", stream, err), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "text/html;charset=UTF-8")
	renderStreamPage(w, stream, tags)
}

func (s *Server) handleRelease(w http.ResponseWriter, r *http.Request, stream string) {
	_, tag, _ := parseReleasePath(r.URL.Path)
	if tag == "" {
		http.NotFound(w, r)
		return
	}

	var (
		wg             sync.WaitGroup
		tags           *APIRelease
		info           *APIReleaseInfo
		qualifiers     *QualifiersResponse
		tagsErr        error
		infoErr        error
		qualifiersErr  error
	)

	wg.Add(3)
	go func() {
		defer wg.Done()
		tags, tagsErr = s.client.GetReleaseTags(stream)
	}()
	go func() {
		defer wg.Done()
		info, infoErr = s.client.GetReleaseInfo(stream, tag)
	}()
	go func() {
		defer wg.Done()
		qualifiers, qualifiersErr = s.client.GetQualifiers(tag)
	}()
	wg.Wait()

	if tagsErr != nil {
		log.Printf("Error fetching tags for %s: %v", stream, tagsErr)
	}
	if infoErr != nil {
		log.Printf("Error fetching release info for %s/%s: %v", stream, tag, infoErr)
	}
	if qualifiersErr != nil {
		log.Printf("Error fetching qualifiers for %s: %v", tag, qualifiersErr)
	}

	// The RC API response includes the predecessor in changeLogJson.from
	var predecessor string
	if info != nil && info.ChangeLogJson != nil && info.ChangeLogJson.From.Name != "" {
		predecessor = info.ChangeLogJson.From.Name
	}
	if from := r.URL.Query().Get("from"); from != "" {
		predecessor = from
	}

	var changelog *ChangelogResponse
	var changelogPending bool
	if predecessor != "" {
		var changelogErr error
		changelog, changelogErr = s.client.GetChangelog(predecessor, tag, "amd64")
		if changelogErr != nil {
			if changelogErr == ErrChangelogPending {
				changelogPending = true
			}
			log.Printf("Error fetching changelog %s -> %s: %v", predecessor, tag, changelogErr)
		}
	}

	w.Header().Set("Content-Type", "text/html;charset=UTF-8")
	renderReleasePage(w, tag, stream, predecessor, tags, info, qualifiers, changelog, changelogPending)
}

// parseReleasePath extracts stream and tag from paths like:
//
//	/releasestream/4-stable                       -> ("4-stable", "", false)
//	/releasestream/4-stable/release/4.18.51       -> ("4-stable", "4.18.51", true)
func parseReleasePath(path string) (stream, tag string, hasRelease bool) {
	path = strings.TrimPrefix(path, "/releasestream/")
	path = strings.TrimSuffix(path, "/")

	if idx := strings.Index(path, "/release/"); idx >= 0 {
		return path[:idx], path[idx+len("/release/"):], true
	}

	return path, "", false
}
