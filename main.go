package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// Store the graph of fedi instances in an adjacency list.  Note some values may have large length, so we store indices here and a mapping elsewhere for the fedi domain.

var (
	id_to_domain = []string {
		"hachyderm.io",
	}
	domain_to_id = map[string]int {
		"hachyderm.io": 0,
	}
	fedigraph = map[int][]int{
	}
)

func main() {
	visited := map[int]bool {}
	to_visit := []int { 0 }

	for len(to_visit) != 0 {
		var next int
		next, to_visit = pop(to_visit)
		domain := id_to_domain[next]

		// fetch the peer list
		fmt.Printf("fetching peers for %q\n", domain)
		url := fmt.Sprintf("https://%s/api/v1/instance/peers", domain)
		res, err := http.Get(url)
		if err != nil {
			fmt.Printf("failed to get peer list from %q: %s\n", domain, err)
			continue
		}
		defer res.Body.Close()
		body, err := io.ReadAll(res.Body)
		if err != nil {
			fmt.Printf("failed to read body of response from %q: %s\n", domain, err)
			continue
		}

		// parse from json
		var results []string
		err = json.Unmarshal(body, &results)
		if err != nil {
			fmt.Printf("failed to parse json response from %q: %s\n", domain, err)
		}

		for _, r := range results {
			id, ok := domain_to_id[r]
			if !ok {
				id_to_domain = append(id_to_domain, r)
				domain_to_id[r] = len(id_to_domain) - 1
				id = domain_to_id[r]
			}

			// add to adjacency list
			_, ok = fedigraph[next]
			if !ok {
				fedigraph[next] = make([]int, 0)
			}
			fedigraph[next] = append(fedigraph[next], id)

			// if we haven't visited it, add it to the list to visit
			_, ok = visited[id]; if !ok {
				to_visit = append(to_visit, id)
			}
		}
		visited[next] = true
	}
	f, err := json.MarshalIndent(fedigraph, "", "  ")
	if err != nil {
		fmt.Printf("failed to marshal fedigraph: %s", err)
	}
	err = os.WriteFile("fedigraph.json", f, 0644)
	if err != nil {
		fmt.Printf("failed to write file: %s", err)
	}
}

func pop(a []int) (int, []int) {
	return a[len(a)-1],a[:len(a)-1]
}
