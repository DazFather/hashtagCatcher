package main

import (
	"strings"
	"sort"
	"time"
)

// ChatInfo rapresent the informations collected in a chat
type ChatInfo struct {
	hashtags map[string]int // Map hashtag -> times used
	resetter *time.Ticker   // function that will loop causing the hashtag to reset
	stop     bool
}

// Trending allows to get the firsts 'cap' most used hashtags
func (i ChatInfo) Trending(cap int) (trend []string) {
	var tags map[string]int = i.hashtags
	if tags == nil {
		return
	}

	// Sort the trending hashtag
	trend = make([]string, 0, len(tags))
	for tag := range tags {
		trend = append(trend, tag)
	}
	sort.SliceStable(trend, func(i, j int) bool {
		return tags[trend[i]] > tags[trend[j]]
	})

	// Take the first 10 results
	if len(trend) > cap {
		trend = trend[:cap]
	}
	return
}

// Save one or more tags on the hashtags map increasing the used conuter
func (i *ChatInfo) Save(tags ...string) {
	if len(tags) == 0 {
		return
	}

	if i.hashtags == nil {
		i.hashtags = make(map[string]int, len(tags))
	}

	for _, tag := range tags {
		i.hashtags[strings.ToLower(tag)]++
	}
}

// SetAutoReset set a time interval and call given function before reset
func (i *ChatInfo) SetAutoReset(interval time.Duration, beforeReset func(ChatInfo)) {
	if i.resetter != nil {
		i.resetter.Reset(interval)
	} else {
		i.resetter = time.NewTicker(interval)
	}

	go func() {
		defer i.resetter.Stop()

		for range i.resetter.C {
			if i.stop {
				i.stop = false
				break
			}
			go beforeReset(*i)
			i.hashtags = nil
		}

	}()

	return
}

// StopAutoReset stops auto-reset mode and clear saved hashtags
func (i *ChatInfo) StopAutoReset() {
	i.stop = true
	i.hashtags = nil
}
