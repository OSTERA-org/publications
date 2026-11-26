// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 OSTERA

package pub

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/goccy/go-yaml"
)

// The commands only need a short name, such as jcs, or a Working Group
// id, such as jss. They look up everything else on wg.ostera.org, and
// suggest it.

// lookupSiteURL is where work items are looked for (wgSiteURL, except in
// the tests), and hubDataURL is the list of Working Groups that the home
// page of wg.ostera.org shows.
var (
	lookupSiteURL = wgSiteURL
	hubDataURL    = "https://raw.githubusercontent.com/OSTERA-org-Working-Groups/ostera-org-working-groups.github.io/main/docs/_data/working_groups.yml"
)

// lookupClient is used to look for a work item on every Working Group
// site at once, so it gives up sooner than the client that downloads a
// document.
var lookupClient = &http.Client{Timeout: 15 * time.Second}

// hubWG is a Working Group, as the home page of wg.ostera.org lists it.
type hubWG struct {
	ID      string `yaml:"id"`
	Name    string `yaml:"name"`
	Abbr    string `yaml:"abbr"`
	Website bool   `yaml:"website"`
}

// displayName is the name used in registry.yaml, such as "JSON Signature
// Scheme (JSS)".
func (w hubWG) displayName() string {
	if w.Abbr != "" && !strings.Contains(w.Name, "("+w.Abbr+")") {
		return w.Name + " (" + w.Abbr + ")"
	}
	return w.Name
}

// loadHub reads the list of Working Groups.
func loadHub() ([]hubWG, error) {
	data, err := download(hubDataURL)
	if err != nil {
		return nil, err
	}
	var list []hubWG
	if err := yaml.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("cannot read %s: %v", hubDataURL, err)
	}
	return list, nil
}

// findWorkItem looks for the work item with this short name on the site
// of every Working Group, both those on wg.ostera.org and those in
// registry.yaml. It returns the work item when exactly one Working Group
// has it.
func findWorkItem(short string, reg *Registry) (workItemURL, *wgItem, error) {
	ids := map[string]bool{}
	if hub, err := loadHub(); err == nil {
		for _, w := range hub {
			if w.Website && idRE.MatchString(w.ID) {
				ids[w.ID] = true
			}
		}
	}
	if reg != nil {
		for id := range reg.WorkingGroups {
			ids[id] = true
		}
	}
	if len(ids) == 0 {
		return workItemURL{}, nil, fmt.Errorf("the list of Working Groups on wg.ostera.org could not be read")
	}

	type hit struct {
		target workItemURL
		item   *wgItem
	}
	var (
		mu   sync.Mutex
		hits []hit
		wg   sync.WaitGroup
	)
	for id := range ids {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			t := workItemURL{site: lookupSiteURL, wg: id, id: short}
			data, err := downloadWith(lookupClient, t.registryURL())
			if err != nil {
				return
			}
			var item *wgItem
			if err := json.Unmarshal(data, &item); err != nil || item == nil {
				return
			}
			mu.Lock()
			hits = append(hits, hit{t, item})
			mu.Unlock()
		}(id)
	}
	wg.Wait()

	switch len(hits) {
	case 0:
		return workItemURL{}, nil, fmt.Errorf("no Working Group on %s has a work item called %s", lookupSiteURL, short)
	case 1:
		return hits[0].target, hits[0].item, nil
	}
	var where []string
	for _, h := range hits {
		where = append(where, h.target.page())
	}
	sort.Strings(where)
	return workItemURL{}, nil, fmt.Errorf("more than one Working Group has a work item called %s: %s; give its page with URL= instead", short, strings.Join(where, ", "))
}

// workItemFor turns what the person gave (a work item page, a short name,
// or nothing, in which case it asks for the short name) into the work
// item. With required set, it is an error when the work item cannot be
// found; otherwise, it says so, and the short name is still returned.
func workItemFor(arg string, reg *Registry, ask *asker, required bool) (workItemURL, *wgItem, error) {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		arg = ask.text("Short name of the work item, such as jcs", "")
		if arg == "" {
			return workItemURL{}, nil, fmt.Errorf("no short name was given; nothing was changed")
		}
	}
	if !strings.Contains(arg, "/") && !idRE.MatchString(arg) {
		return workItemURL{}, nil, fmt.Errorf("%q is not a short name: use lowercase letters and digits, starting with a letter; nothing was changed", arg)
	}
	if !idRE.MatchString(arg) {
		target, err := parseWorkItemURL(arg)
		if err != nil {
			return workItemURL{}, nil, err
		}
		item, err := fetchWorkItem(target.registryURL())
		if err != nil {
			if required {
				return workItemURL{}, nil, err
			}
			fmt.Fprintf(ask.out, "(The work item page could not be read, so only its short name is suggested.)\n\n")
			return target, nil, nil
		}
		fmt.Fprintf(ask.out, "Found %s\n  %s\n\n", item.Name, target.page())
		return target, item, nil
	}
	fmt.Fprintf(ask.out, "Looking for %s on %s ...\n", arg, lookupSiteURL)
	target, item, err := findWorkItem(arg, reg)
	if err != nil {
		if required {
			return workItemURL{}, nil, err
		}
		fmt.Fprintf(ask.out, "(%s, so there are no suggestions.)\n\n", strings.ToUpper(err.Error()[:1])+err.Error()[1:])
		return workItemURL{id: arg}, nil, nil
	}
	fmt.Fprintf(ask.out, "Found %s, in Working Group %s\n  %s\n\n", item.Name, target.wg, target.page())
	return target, item, nil
}

// wgNameFor suggests the name of a Working Group: from the list on
// wg.ostera.org, or else from the website settings of its repository.
func wgNameFor(id string) string {
	if hub, err := loadHub(); err == nil {
		for _, w := range hub {
			if w.ID == id && w.Name != "" {
				return w.displayName()
			}
		}
	}
	if cfg, err := download(fmt.Sprintf(wgConfigURL, id)); err == nil {
		if m := shortTitleRE.FindSubmatch(cfg); m != nil {
			return strings.TrimSpace(string(m[1]))
		}
	}
	return ""
}
