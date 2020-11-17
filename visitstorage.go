package scrapingo

import "sync"

type VisitStorage interface {
	IsVisited(uint64) bool
	Visited(uint64)
}

type HasStorage struct {
	rw         *sync.RWMutex
	visitedmap map[uint64]bool
}

//scrapingo默認使用去重儲存
func defaultHasStorage() *HasStorage {
	return &HasStorage{rw: &sync.RWMutex{}, visitedmap: make(map[uint64]bool)}
}

//實現VisitStorage interface IsVisited()
//將URL轉成的哈希值判斷是否有重複值
func (h *HasStorage) IsVisited(reqId uint64) bool {
	h.rw.RLock()
	defer h.rw.RUnlock()
	if ok := h.visitedmap[reqId]; ok {
		return true
	}
	return false
}

//實現VisitStorage interface Visited()
//將URL轉成的哈希值並且儲存
func (h *HasStorage) Visited(reqId uint64) {
	h.rw.Lock()
	defer h.rw.Unlock()
	h.visitedmap[reqId] = true
}
