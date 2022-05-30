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
