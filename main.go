package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
)

// Store the graph of fedi instances in an adjacency list.  Note some values may have large length, so we store indices here and a mapping elsewhere for the fedi domain.

type Graph struct {
	IdToDomain []string
	Adjacency  map[int][]int
	m          sync.RWMutex `json:"-"`
}

var (
	graph = Graph{
		IdToDomain: []string{"hachyderm.io"},
		Adjacency:  map[int][]int{},
	}
	domainToId = map[string]int{
		"hachyderm.io": 0,
	}
	toVisit = []int{0}
	toVisitSet = map[int]bool{0: true}
	visited = map[int]bool{}
)

func main() {
	for len(toVisit) != 0 {
		graph.m.RLock()
		var next int
		next, toVisit = pop(toVisit)
		domain := graph.IdToDomain[next]
		graph.m.RUnlock()

		// fetch the peer list
		fmt.Printf("fetching peers for %q (%d / %d)\n", domain, len(visited), len(toVisit))
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

		graph.m.Lock()
		for _, r := range results {
			id, ok := domainToId[r]
			if !ok {
				graph.IdToDomain = append(graph.IdToDomain, r)
				domainToId[r] = len(graph.IdToDomain) - 1
				id = domainToId[r]
			}

			// add to adjacency list
			_, ok = graph.Adjacency[next]
			if !ok {
				graph.Adjacency[next] = make([]int, 0)
			}
			graph.Adjacency[next] = append(graph.Adjacency[next], id)

			// if we haven't visited it, and we don't plan to yet, add it to the list to visit
			_, visitedOk := visited[id]
			_, toVisitOk := toVisitSet[id]
			if !visitedOk && !toVisitOk {
				toVisit = append(toVisit, id)
				toVisitSet[id] = true
			}
		}
		visited[next] = true
		graph.m.Unlock()
		go dumpGraph()
	}
}

func dumpGraph() {
	graph.m.RLock()
	f, err := json.MarshalIndent(graph, "", "  ")
	graph.m.RUnlock()
	if err != nil {
		fmt.Printf("failed to marshal fedigraph: %s", err)
	}
	if err = os.WriteFile("fedigraph.json", f, 0644); err != nil {
		fmt.Printf("failed to write file: %s", err)
	}
}

func pop(a []int) (int, []int) {
	popped, rest := a[len(a)-1], a[:len(a)-1]
	delete(toVisitSet, popped)
	return popped, rest
}
