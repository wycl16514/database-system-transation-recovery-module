前面一节我们完成了用于实现系统恢复的日志，本节我们看看如何基于日志内容实现系统恢复。我们将设计一个系统恢复管理器，它在系统启动时读取日志内容，根据读到的日志对数据进行恢复，由于所谓“恢复”其实是交易的回滚，因此我们首先实现交易对象，前面为了测试方便，我们简单的提供了交易对象的几个简单接口，这里我们将实现一个逻辑完整的交易对象，只不过我们暂时忽略其并发管理逻辑，并发功能我们将在后面的章节进行实现。

首先我们先了解交易对象的基本结构：
![请添加图片描述](https://img-blog.csdnimg.cn/5e2700b97e854e2ea5a5c6b04c4d4090.png)

这里我们先忽略并发管理，它将在后一节进行针对性的研究，我们首先实现Transation,BufferList,和RecoverMgr。前面章节中我们使用Buffer对象来实现数据写入缓存页面或者存入磁盘，而Transation其实是对Buffer提供接口的封装和调用，它除了支持数据读写功能外，还在此基础上提供了并发控制，恢复控制等功能，后面其他模块都必须通过Transation对象来实现数据的写入和读取，首先我们在interface.go中增加一个常量定义：
```
const (
	UINT64_LENGTH = 8
	END_OF_FILE = -1
)
```
接着增加一个buffer_list.go，它用来记录或快速查询当前被pin的内存页面，其内容如下：
```
package tx

import (
	fm "file_manager"
	"fmt"
	bm "buffer_manager"
)

type BufferList struct {
	buffers map[*fm.BlockId]*bm.Buffer 
	buffer_mgr *bm.BufferManager
	pins []*fm.BlockId
}

func NewBufferList(buffer_mgr *bm.BufferManager) *BufferList {
	buffer_list := &BufferList{
		buffer_mgr : buffer_mgr,
		buffers: make(map[*fm.BlockId]*bm.Buffer ),
		pins: []fm.BlockId,
	}

	return buffer_list
}

func (b *BufferList) get_buffer(blk *fm.BlockId) *bm.Buffer {
	buff, _ := b.buffers[blk]
	return buff 
}

func (b *BufferList) Pin(blk *fm.BlockId) error{
	//一旦一个内存页被pin后，将其加入map进行追踪管理
	buff, err := b.buffer_mgr.Pin(blk)
	if err != nil {
		return err
	}
    s.buffers[blk] = buff
	b.pins = append(b.pins, blk)
	return nil 
}

func (b *BufferList)Unpin(blk *fm.BlockId) {
	buffer, ok := b.buffers[blk]
	if !ok {
		return 
	}

	b.buffer_mgr.Unpin(blk)
	for idx, pinned_blk := range b.pins {
		if pinned_blk == blk {
			b.pins = append(s.pins[:idx], s.pins[idx+1]...)
			break
		}
	}

	delete(s.buffers, blk)
}

func (b *BufferList)UnpinAll() {
	for _, blk in range b.pins {
		buffer := s.buffers[blk]
		s.buffer_mgr.Unpin(buffer)
	}

    s.buffers = make(map[*fm.BlockId]*bm.Buffer)
    s.pins = make([]*fm.BlockId)
}
```
下面我们看看交易对象的实现，增加transation.go，添加代码如下：
```
package tx

import (
	fm "file_manager"
	"fmt"
	lg "log_manager"
	bm "buffer_manager"
	"sync"
	"errors"
)

var tx_num_mu sync.Mutex
next_tx_num := int32(0)

func NxtTxNum() int32{
	tx_num_mu.Lock()
	defer tx_num_mu.Unlock()

	next_tx_num = next_tx_num + 1

	return next_tx_num
}

type  Transation struct{
    //concur_mgr  ConcurrentMgr*
	//recovery_mgr RecorveryMgr* 
    file_manager *fm.FileManager
	log_manager *lg.LogManager 
	buffer_manager *bm.BufferManager
	my_buffers  *BufferList 
    tx_num      int32
}

func NewTransation(file_manager *fm.FileManager, log_manager *lg.LogManager, 
	buffer_manager bm.BufferManager) *Transation {
		tx := &Transation {
			//创建同步管理器
			//创建恢复管理器
			file_manager: file_manager,
			log_manager: log_manager,
			buffer_manager: buffer_manager,
			my_buffers: NewBufferList(buffer_manager), 
		}

		return tx
}

func (t *Transation)Commit() {
	//调用恢复管理器执行commit
	//t.recovery_mgr.Commit()

	r := fmt.Sprintf("transation %d  committed", t.tx_num)
	fmt.Println(r)
	//释放同步管理器
	t.my_buffers.UnpinAll()
}

func (t *Transation) Rollback() {
	//调用恢复管理器rollback
	//t.recovery_mft.Rollback()
	r := fmt.Sprintf("transation %d roll back", t.tx_num)
	//释放同步管理器
	t.my_buffers.UnpinAll()
}

func(t *Transation)Recover() {
	//系统启动时会在所有交易执行前执行该函数
	t.bm.FlushAll(t.tx_num)
	//调用回复管理器的recover接口
	//t.recovery_mgr.Recover()
}

func (t *Transation)Pin(blk *fm.BlockId) {
	t.my_buffers.Pin(blk)
}

func (t *Transation) Unpin(blk *fm.BlockId) {
	t.my_buffers.Unpin(blk)
}

func (t *Transation) buffer_no_exist(blk *fm.BlockId) error{
	err_s := fmt.Sprintf("No buffer found for given blk : %d with file name: %s\n", 
	blk.Number(), blk.FileName())
	err := errors.New(err_s)
	return err 
}

func (t *Transation) GetInt(blk *fm.BlockId, offset uint64) int64, error {
	//调用同步管理器加s锁
	//t.concur_mgr.Slock(blk)

	buff := t.my_buffers.get_buffer(blk)
	if buff == nil {
		return -1, t.buffer_no_exist(blk) 
	}

	return int64(buff.Contents.GetInt(offset)), nil 
}

func(t *Transation) GetString(blk *fm.BlockId, offset uint64) string, error {
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

	lsn := 0
	if okToLog {
		//调用恢复管理器的SetInt方法
		//lsn = t.recovery_mgr.SetInt(buff, offset, val)
	}

	p = buff.Contents()
	p.SetInt(offset, uint64(val))
	buff.SetModified(t.tx_num, lsn)
}

func (t *Transation) SetString(blk *fm.BlockId, offset uint64, val string, okToLog bool) error {
	//使用同步管理器加x锁
	//t.concur_mgr.Xlock(blk)
	
	buff := t.my_buffers.get_buffer(blk)
	if buff == nil {
		return t.buffer_no_exist(blk)
	}

	lsn := 0
	if okToLog {
		//调用恢复管理器SetString方法
		//lsn = t.recovery_mgr.SetString(buff, offset, val)
	}

	p := buff.Contents()
	p.SetString(offset, val)
	buff.SetModified(t.tx_num, lsn)
}

func (t *Transation) Size(file_name string) uint64 {
	//调用同步管理器加锁
	//dummy_blk := fm.NewBlockId(file_name, END_OF_FILE)
	//t.concur_mgr.Slock(dummy_blk)
	return t.file_manager.Size(file_name)
}

func (t *Transation)Append(file_name string) *fm.BlockId{
	//调用同步管理器加锁
	//dummy_blk := fm.NewBlockId(file_name, END_OF_FILE)
	//t.concur_mgr.Xlock(dummy_blk)
	blk, err := t.file_manager.Append(file_name)
	if err != nil {
		return nil 
	}

	return blk 
}

func (t *Transation) BlockSize() uint64{
	return t.file_manager.BlockSize()
}

func (t *Transation) AvailableBuffers() uint64{
	return t.buffer_manager.Available()
}


```
由于交易对象在执行写入或读出时需要根据并发情况加相应的锁，而且它在写入数据时还需要调用恢复管理器记录写入状况以便未来执行恢复操作，但是并发管理器和恢复管理器目前尚未实现，因此我们在调用他们的接口时先注释掉。下面我们看看恢复管理器的实现，一旦完成恢复管理器的代码后，我们再将上面涉及到恢复管理器的注释进行返注释。

下面我们再看看恢复管理器的实现，添加recovery_mgr.go，添加代码如下：
```
package tx

import (
	bm "buffer_manager"
	fm "file_manager"
	lg "log_manager"
)

type RecoveryManager struct {
	log_manager    *lg.LogManager
	buffer_manager *bm.BufferManager
	tx             *Transation
	tx_num         int32
}

func NewRecoveryManager(tx *Transation, tx_num int32, log_manager *lg.LogManager,
	buffer_manager *bm.BufferManager) *RecoveryManager {
	recovery_mgr := &RecoveryManager{
		tx:             tx,
		log_manager:    log_manager,
		buffer_manager: buffer_manager,
	}

	p := fm.NewPageBySize(32)
	p.SetInt(0, uint64(START))
	p.SetInt(8, uint64(tx_num))
	start_record := NewStartRecord(p, log_manager)
	start_record.WriteToLog()

	return recovery_mgr
}

func (r *RecoveryManager) Commit() error {
	r.buffer_manager.FlushAll(r.tx_num)
	lsn, err := WriteCommitkRecordLog(r.log_manager, uint64(r.tx_num))
	if err != nil {
		return err
	}

	r.log_manager.FlushByLSN(lsn)
	return nil
}

func (r *RecoveryManager) Rollback() error {
	r.doRollback()
	r.buffer_manager.FlushAll(r.tx_num)
	lsn, err := WriteRollBackLog(r.log_manager, uint64(r.tx_num))
	if err != nil {
		return err
	}

	r.log_manager.FlushByLSN(lsn)
	return nil
}

func (r *RecoveryManager) Recover() error {
	r.doRecover()
	r.buffer_manager.FlushAll(r.tx_num)
	lsn, err := WriteCheckPointToLog(r.log_manager)
	if err != nil {
		return err
	}

	r.log_manager.FlushByLSN(lsn)
	return nil
}

func (r *RecoveryManager) SetInt(buffer *bm.Buffer, offset uint64, new_val int64) (uint64, error) {
	old_val := buffer.Contents().GetInt(offset)
	blk := buffer.Block()
	buffer.Contents().SetInt(offset, uint64(new_val))
	return WriteSetIntLog(r.log_manager, uint64(r.tx_num), blk, offset, old_val)
}

func (r *RecoveryManager) SetString(buffer *bm.Buffer, offset uint64, new_val string) (uint64, error) {
	old_val := buffer.Contents().GetString(offset)
	blk := buffer.Block()
	buffer.Contents().SetString(offset, new_val)
	return WriteSetStringLog(r.log_manager, uint64(r.tx_num), blk, offset, old_val)
}

func (r *RecoveryManager) CreateLogRecord(bytes []byte) LogRecordInterface {
	p := fm.NewPageByBytes(bytes)
	switch RECORD_TYPE(p.GetInt(0)) {
	case CHECKPOINT:
		return NewCheckPointRecord()
	case START:
		return NewStartRecord(p, r.log_manager)
	case COMMIT:
		return NewCommitkRecordRecord(p)
	case ROLLBACK:
		return NewRollBackRecord(p)
	case SETINT:
		return NewSetIntRecord(p)
	case SETSTRING:
		return NewSetStringRecord(p)
	default:
		panic("Unknow log interface")
	}
}

func (r *RecoveryManager) doRollback() {
	iter := r.log_manager.Iterator()
	for iter.HasNext() {
		rec := iter.Next()
		log_record := r.CreateLogRecord(rec)
		if log_record.TxNumber() == uint64(r.tx_num) {
			if log_record.Op() == START {
				return
			}

			log_record.Undo(r.tx)
		}
	}
}

func (r *RecoveryManager) doRecover() {
	finishedTxs := make(map[uint64]bool)
	iter := r.log_manager.Iterator()
	for iter.HasNext() {
		rec := iter.Next()
		log_record := r.CreateLogRecord(rec)
		if log_record.Op() == CHECKPOINT {
			return
		}
		if log_record.Op() == COMMIT || log_record.Op() == ROLLBACK {
			finishedTxs[log_record.TxNumber()] = true
		}
		existed, _ := finishedTxs[log_record.TxNumber()]
		if existed {
			log_record.Undo(r.tx)
		}
	}
}

```
完成上面代码后，我们记得取消掉transation.go里面关于恢复管理器的注释，为了检测我们的代码基本逻辑是否正确，我们在main.go中拟写如下代码：
```
package main

import (
	//"encoding/binary"
	fm "file_manager"
	lm "log_manager"

	bmg "buffer_manager"

	"fmt"

	"tx"
)

func main() {
	file_manager, _ := fm.NewFileManager("txtest", 400)
	log_manager, _ := lm.NewLogManager(file_manager, "logfile")
	buffer_manager := bmg.NewBufferManager(file_manager, log_manager, 3)

	tx1 := tx.NewTransation(file_manager, log_manager, buffer_manager)
	blk := fm.NewBlockId("testfile", 1)
	tx1.Pin(blk)
	//设置log为false，因为一开始数据没有任何意义，因此不能进行日志记录
	tx1.SetInt(blk, 80, 1, false) 
	tx1.SetString(blk, 40, "one", false)
	tx1.Commit() //执行回滚操作后，数据会还原到这里写入的内容

	tx2 := tx.NewTransation(file_manager, log_manager, buffer_manager)
	tx2.Pin(blk)
	ival, _ := tx2.GetInt(blk, 80)
	sval, _ := tx2.GetString(blk, 40)
	fmt.Println("initial value at location 80 = ", ival)
	fmt.Println("initial value at location 40 = ", sval)
	new_ival := ival + 1
	new_sval := sval + "!"
	tx2.SetInt(blk, 80, new_ival, true)
	tx2.SetString(blk, 40, new_sval, true)
	tx2.Commit() //尝试写入新的数据

	tx3 := tx.NewTransation(file_manager, log_manager, buffer_manager)
	tx3.Pin(blk)
	ival, _ = tx3.GetInt(blk, 80)
	sval, _ = tx3.GetString(blk, 40)
	fmt.Println("new ivalue at location 80: ", ival)
	fmt.Println("new svalue at location 40: ", sval)
	tx3.SetInt(blk, 80, 999, true)
	ival, _ = tx3.GetInt(blk, 80)
	//写入数据后检查是否写入正确
	fmt.Println("pre-rollback ivalue at location 80: ", ival)
	tx3.RollBack() //执行回滚操作，并确定回滚到第一次写入内容

	tx4 := tx.NewTransation(file_manager, log_manager, buffer_manager)
	tx4.Pin(blk)
	ival, _ = tx4.GetInt(blk, 80)
	fmt.Println("post-rollback at location 80 = ", ival)
	tx4.Commit() //执行到这里时，输出内容应该与第一次写入内容相同

}
```
完成后我们执行上面代码得到输出如下：
![请添加图片描述](https://img-blog.csdnimg.cn/3604ab9e747241938e63971201d42211.png)
从输出结果看，我们先在给定位置写入数值1，然后再写入数组2,999，最后调用RollBack()执行回滚，然后再取出同样位置的数据进行打印，通过输出可以看到，RollBack执行后，给定位置的数值变成了最开始我们输入的数值，如此看来，恢复管理器的基本逻辑应该是正确的。更详细的调试演示请在b站搜索Coding迪斯尼，[更多干货](http://m.study.163.com/provider/7600199/index.htm?share=2&shareId=7600199)：http://m.study.163.com/provider/7600199/index.htm?share=2&shareId=7600199
