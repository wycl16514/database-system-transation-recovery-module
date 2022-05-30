package tx

import (
	bm "buffer_manager"
	fm "file_manager"
)

type BufferList struct {
	buffers    map[*fm.BlockId]*bm.Buffer
	buffer_mgr *bm.BufferManager
	pins       []*fm.BlockId
}

func NewBufferList(buffer_mgr *bm.BufferManager) *BufferList {
	buffer_list := &BufferList{
		buffer_mgr: buffer_mgr,
		buffers:    make(map[*fm.BlockId]*bm.Buffer),
		pins:       make([]*fm.BlockId, 0),
	}

	return buffer_list
}

func (b *BufferList) get_buffer(blk *fm.BlockId) *bm.Buffer {
	buff, _ := b.buffers[blk]
	return buff
}

func (b *BufferList) Pin(blk *fm.BlockId) error {
	//一旦一个内存页被pin后，将其加入map进行追踪管理
	buff, err := b.buffer_mgr.Pin(blk)
	if err != nil {
		return err
	}
	b.buffers[blk] = buff
	b.pins = append(b.pins, blk)
	return nil
}

func (b *BufferList) Unpin(blk *fm.BlockId) {
	buff, ok := b.buffers[blk]
	if !ok {
		return
	}

	b.buffer_mgr.Unpin(buff)
	for idx, pinned_blk := range b.pins {
		if pinned_blk == blk {
			b.pins = append(b.pins[:idx], b.pins[idx+1:]...)
			break
		}
	}

	delete(b.buffers, blk)
}

func (b *BufferList) UnpinAll() {
	for _, blk := range b.pins {
		buffer := b.buffers[blk]
		b.buffer_mgr.Unpin(buffer)
	}

	b.buffers = make(map[*fm.BlockId]*bm.Buffer)
	b.pins = make([]*fm.BlockId, 0)
}
