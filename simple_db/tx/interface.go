package tx

import (
	fm "file_manager"
)

type TransationInterface interface {
	Commit()
	RollBack()
	Recover()
	Pin(blk *fm.BlockId)
	UnPin(blk *fm.BlockId)
	GetInt(blk *fm.BlockId, offset uint64) (int64, error)
	GetString(blk *fm.BlockId, offset uint64) (string, error)
	SetInt(blk *fm.BlockId, offset uint64, val int64, okToLog bool) error
	SetString(blk *fm.BlockId, offset uint64, val string, okToLog bool) error
	AvailableBuffers() uint64
	Size(filename string) uint64
	Append(filename string) *fm.BlockId
	BlockSize() uint64
}

type RECORD_TYPE uint64

const (
	CHECKPOINT RECORD_TYPE = iota
	START
	COMMIT
	ROLLBACK
	SETINT
	SETSTRING
)

const (
	UINT64_LENGTH = 8
	END_OF_FILE   = -1
)

type LogRecordInterface interface {
	Op() RECORD_TYPE             //返回记录的类别
	TxNumber() uint64            //对应交易的号码
	Undo(tx TransationInterface) //回滚操作
	ToString() string            //获得记录的字符串内容
}
