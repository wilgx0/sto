package sto

import (
	"sort"
	"sync"
	"time"
)

func New(opts ...Option) *Cron {
	c := &Cron{
		entries:   nil,
		chain:     NewChain(),
		add:       make(chan *Entry),
		stop:      make(chan struct{}),
		snapshot:  make(chan chan []Entry),
		remove:    make(chan EntryID),
		running:   false,
		runningMu: sync.Mutex{},
		location:  time.Local,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

type Cron struct {
	entries   []*Entry
	chain     Chain
	stop      chan struct{}
	add       chan *Entry
	remove    chan EntryID
	snapshot  chan chan []Entry
	running   bool
	runningMu sync.Mutex
	location  *time.Location
	nextID    EntryID
	jobWaiter sync.WaitGroup
}

// now returns current time in c location
func (c *Cron) now() time.Time {
	return time.Now().In(c.location)
}

// Start the cron scheduler in its own goroutine, or no-op if already started.
func (c *Cron) Start() {
	c.runningMu.Lock()
	defer c.runningMu.Unlock()
	if c.running {
		return
	}
	c.running = true
	go c.run()
}

func (c *Cron) run() {
	now := c.now()
	for {
		sort.Sort(byTime(c.entries))
		var timer *time.Timer
		if len(c.entries) == 0 || c.entries[0].Next.IsZero() {
			// If there are no entries yet, just sleep - it still handles new entries
			// and stop requests.
			timer = time.NewTimer(100000 * time.Hour)
		} else {
			timer = time.NewTimer(c.entries[0].Next.Sub(now))
		}

		for {
			select {
			case now = <-timer.C:
				for len(c.entries) > 0 {
					e := c.entries[0]
					if e.Next.After(now) || e.Next.IsZero() {
						break
					}
					c.entries = c.entries[1:]
					c.startJob(e.WrappedJob, e.Payload)
				}

			case newEntry := <-c.add:
				timer.Stop()
				c.entries = append(c.entries, newEntry)

			case replyChan := <-c.snapshot:
				replyChan <- c.entrySnapshot()
				continue
			case <-c.stop:
				timer.Stop()
				return

			case id := <-c.remove:
				timer.Stop()
				now = c.now()
				c.removeEntry(id)
			}

			break
		}
	}
}

// Entries returns a snapshot of the cron entries.
func (c *Cron) Entries() []Entry {
	c.runningMu.Lock()
	defer c.runningMu.Unlock()
	if c.running {
		replyChan := make(chan []Entry, 1)
		c.snapshot <- replyChan
		return <-replyChan
	}
	return c.entrySnapshot()
}

func (c *Cron) Filter(fn func(payload interface{}) bool) (entrys []Entry) {
	entries := c.Entries()
	for _, item := range entries {
		if fn(item.Payload) {
			entrys = append(entrys, item)
		}
	}

	return entrys
}

// Location gets the time zone location
func (c *Cron) Location() *time.Location {
	return c.location
}

// Entry returns a snapshot of the given entry, or nil if it couldn't be found.
func (c *Cron) Entry(id EntryID) Entry {
	for _, entry := range c.Entries() {
		if id == entry.ID {
			return entry
		}
	}
	return Entry{}
}

func (c *Cron) AddFunc(next time.Time, cmd func(payload interface{}), payload interface{}) EntryID {
	return c.AddJob(next, FuncJob(cmd), payload)
}

func (c *Cron) AddJob(next time.Time, cmd Job, payload interface{}) EntryID {
	c.runningMu.Lock()
	defer c.runningMu.Unlock()
	c.nextID++
	entry := &Entry{
		ID:         c.nextID,
		Next:       next,
		WrappedJob: c.chain.Then(cmd, payload),
		Payload:    payload,
	}
	if !c.running {
		c.entries = append(c.entries, entry)
	} else {
		c.add <- entry
	}
	return entry.ID
}

func (c *Cron) Remove(id EntryID) {
	c.runningMu.Lock()
	defer c.runningMu.Unlock()
	if c.running {
		c.remove <- id
	} else {
		c.removeEntry(id)
	}
}

func (c *Cron) entrySnapshot() []Entry {
	var entries = make([]Entry, len(c.entries))
	for i, e := range c.entries {
		entries[i] = *e
	}
	return entries
}

func (c *Cron) startJob(j Job, payload interface{}) {
	c.jobWaiter.Add(1)
	go func() {
		defer c.jobWaiter.Done()
		j.Run(payload)
	}()
}

func (c *Cron) removeEntry(id EntryID) {
	var entries []*Entry
	for _, e := range c.entries {
		if e.ID != id {
			entries = append(entries, e)
		}
	}
	c.entries = entries
}

type byTime []*Entry

func (s byTime) Len() int      { return len(s) }
func (s byTime) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s byTime) Less(i, j int) bool {
	// Two zero times should return false.
	// Otherwise, zero is "greater" than any other time.
	// (To sort it at the end of the list.)
	if s[i].Next.IsZero() {
		return false
	}
	if s[j].Next.IsZero() {
		return true
	}
	return s[i].Next.Before(s[j].Next)
}

type Job interface {
	Run(payload interface{})
}

type FuncJob func(payload interface{})

func (f FuncJob) Run(payload interface{}) { f(payload) }

type EntryID int
type Entry struct {
	ID         EntryID
	Next       time.Time
	WrappedJob Job
	Payload    interface{}
}
