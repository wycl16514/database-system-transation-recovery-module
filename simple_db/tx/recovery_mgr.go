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
