// Copyright 2018 The uranus Authors
// This file is part of the uranus library.
//
// The uranus library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The uranus library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the uranus library. If not, see <http://www.gnu.org/licenses/>.

package feed_test

import (
	"fmt"

	"github.com/UranusBlockStack/uranus/feed"
)

func ExampleFeed_acknowledgedfeeds() {

	var feed feed.Feed
	type ackedfeed struct {
		i   int
		ack chan<- struct{}
	}

	// Consumers wait for feeds on the feed and acknowledge processing.
	done := make(chan struct{})
	defer close(done)
	for i := 0; i < 3; i++ {
		ch := make(chan ackedfeed, 100)
		sub := feed.Subscribe(ch)
		go func() {
			defer sub.Unsubscribe()
			for {
				select {
				case ev := <-ch:
					fmt.Println(ev.i) // "process" the feed
					ev.ack <- struct{}{}
				case <-done:
					return
				}
			}
		}()
	}

	for i := 0; i < 3; i++ {
		acksignal := make(chan struct{})
		n := feed.Send(ackedfeed{i, acksignal})
		for ack := 0; ack < n; ack++ {
			<-acksignal
		}
	}
	// Output:
	// 0
	// 0
	// 0
	// 1
	// 1
	// 1
	// 2
	// 2
	// 2
}
