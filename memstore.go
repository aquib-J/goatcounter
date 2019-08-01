package goatcounter

import (
	"context"
	"sync"

	"zgo.at/goatcounter/bulk"
	"zgo.at/zlog"
)

type ms struct {
	sync.RWMutex
	hits     []Hit
	browsers []Browser
}

var Memstore = ms{}

func (m *ms) Append(hit Hit, browser Browser) {
	m.Lock()
	m.hits = append(m.hits, hit)
	m.browsers = append(m.browsers, browser)
	m.Unlock()
}

func (m *ms) Persist(ctx context.Context) error {
	if len(m.hits) == 0 {
		return nil
	}

	l := zlog.Debug("memstore").Module("memstore")

	m.Lock()
	hits := make([]Hit, len(m.hits))
	browsers := make([]Browser, len(m.browsers))
	copy(hits, m.hits)
	copy(browsers, m.browsers)
	m.hits = []Hit{}
	m.browsers = []Browser{}
	m.Unlock()

	l.Printf("persisting %d hits and %d User-Agents", len(hits), len(browsers))

	ins := bulk.NewInsert(ctx, MustGetDB(ctx),
		"hits", []string{"site", "path", "ref", "ref_params", "ref_original", "created_at"})
	for _, h := range hits {
		h.Defaults(ctx)
		err := h.Validate(ctx)
		if err != nil {
			zlog.Error(err)
			continue
		}

		ins.Values(h.Site, h.Path, h.Ref, h.RefParams, h.RefOriginal, sqlDate(h.CreatedAt))
	}
	err := ins.Finish()
	if err != nil {
		zlog.Error(err)
	}

	l = l.Since("hits done")

	ins = bulk.NewInsert(ctx, MustGetDB(ctx),
		"browsers", []string{"site", "browser", "created_at"})
	for _, b := range browsers {
		b.Defaults(ctx)
		err := b.Validate(ctx)
		if err != nil {
			zlog.Error(err)
			continue
		}

		ins.Values(b.Site, b.Browser, sqlDate(b.CreatedAt))
	}
	err = ins.Finish()
	if err != nil {
		zlog.Error(err)
	}
	l = l.Since("browsers done")

	return nil
}
