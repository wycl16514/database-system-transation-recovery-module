package tx

import (
	bm "buffer_manager"
	"errors"
	fm "file_manager"
	"fmt"
	lg "log_manager"
	"sync"
)

var tx_num_mu sync.Mutex
var next_tx_num = int32(0)

func nextTxNum() int32 {
	tx_num_mu.Lock()
	defer tx_num_mu.Unlock()

	next_tx_num = next_tx_num + 1

	return next_tx_num
}

type Transation struct {
	//concur_mgr  *ConcurrentMgr
	recovery_mgr   *RecoveryManager
	file_manager   *fm.FileManager
	log_manager    *lg.LogManager
	buffer_manager *bm.BufferManager
	my_buffers     *BufferList
	tx_num         int32
}

func NewTransation(file_manager *fm.FileManager, log_manager *lg.LogManager,
	buffer_manager *bm.BufferManager) *Transation {
	tx_num := nextTxNum()
	tx := &Transation{
		file_manager:   file_manager,
		log_manager:    log_manager,
		buffer_manager: buffer_manager,
		my_buffers:     NewBufferList(buffer_manager),
		tx_num:         tx_num,
	}
	//创建同步管理器
	//创建恢复管理器
	tx.recovery_mgr = NewRecoveryManager(tx, tx_num, log_manager, buffer_manager)

	return tx
}

func (t *Transation) Commit() {
	//调用恢复管理器执行commit
	t.recovery_mgr.Commit()

	r := fmt.Sprintf("transation %d  committed", t.tx_num)
	fmt.Println(r)
	//释放同步管理器
	t.my_buffers.UnpinAll()
}

func (t *Transation) RollBack() {
	//调用恢复管理器rollback
	t.recovery_mgr.Rollback()
	r := fmt.Sprintf("transation %d roll back", t.tx_num)
	fmt.Println(r)
	//释放同步管理器
	t.my_buffers.UnpinAll()
}

func (t *Transation) Recover() {
	//系统启动时会在所有交易执行前执行该函数
	t.buffer_manager.FlushAll(t.tx_num)
	//调用回复管理器的recover接口
	t.recovery_mgr.Recover()
}

func (t *Transation) Pin(blk *fm.BlockId) {
	t.my_buffers.Pin(blk)
}

func (t *Transation) UnPin(blk *fm.BlockId) {
	t.my_buffers.Unpin(blk)
}

func (t *Transation) buffer_no_exist(blk *fm.BlockId) error {
	err_s := fmt.Sprintf("No buffer found for given blk : %d with file name: %s\n",
		blk.Number(), blk.FileName())
	err := errors.New(err_s)
	return err
}

func (t *Transation) GetInt(blk *fm.BlockId, offset uint64) (int64, error) {
	//调用同步管理器加s锁
	//t.concur_mgr.Slock(blk)

	buff := t.my_buffers.get_buffer(blk)
	if buff == nil {
		return -1, t.buffer_no_exist(blk)
	}

	return int64(buff.Contents().GetInt(offset)), nil
}

func (t *Transation) GetString(blk *fm.BlockId, offset uint64) (string, error) {
	//调用同步管理器加s锁
	//t.concur_mgr.Slock(blk)

	buff := t.my_buffers.get_buffer(blk)
	if buff == nil {
		return "", t.buffer_no_exist(blk)
	}

	return buff.Contents().GetString(offset), nil
}

func (t *Transation) SetInt(blk *fm.BlockId, offset uint64, val int64, okToLog bool) error {
	//调用同步管理器加x锁
	//t.concur_mgr.Xlock(blk)

	buff := t.my_buffers.get_buffer(blk)
	if buff == nil {
		return t.buffer_no_exist(blk)
	}

	var lsn uint64
	var err error
	if okToLog {
		//调用恢复管理器的SetInt方法
		lsn, err = t.recovery_mgr.SetInt(buff, offset, val)
		if err != nil {
			return err
		}
	}

	p := buff.Contents()
	p.SetInt(offset, uint64(val))
	buff.SetModified(t.tx_num, lsn)
	return nil
}

func (t *Transation) SetString(blk *fm.BlockId, offset uint64, val string, okToLog bool) error {
	//使用同步管理器加x锁
	//t.concur_mgr.Xlock(blk)

	buff := t.my_buffers.get_buffer(blk)
	if buff == nil {
		return t.buffer_no_exist(blk)
	}

	var lsn uint64
	var err error
	if okToLog {
		//调用恢复管理器SetString方法
		lsn, err = t.recovery_mgr.SetString(buff, offset, val)
		if err != nil {
			return err
		}
	}

	p := buff.Contents()
	p.SetString(offset, val)
	buff.SetModified(t.tx_num, lsn)
	return nil
}

func (t *Transation) Size(file_name string) uint64 {
	//调用同步管理器加锁
	//dummy_blk := fm.NewBlockId(file_name, END_OF_FILE)
	//t.concur_mgr.Slock(dummy_blk)
	s, _ := t.file_manager.Size(file_name)
	return s
}

func (t *Transation) Append(file_name string) *fm.BlockId {
	//调用同步管理器加锁
	//dummy_blk := fm.NewBlockId(file_name, END_OF_FILE)
	//t.concur_mgr.Xlock(dummy_blk)
	blk, err := t.file_manager.Append(file_name)
	if err != nil {
		return nil
	}

	return &blk
}

func (t *Transation) BlockSize() uint64 {
	return t.file_manager.BlockSize()
}

func (t *Transation) AvailableBuffers() uint64 {
	return uint64(t.buffer_manager.Available())
}
