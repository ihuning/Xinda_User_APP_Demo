package fragment

import (
	"fmt"
	"math"
	"sync"
	"xindauserbackground/src/filetools"
)

type DivideMethod int

// 三种不同的分片策略
const (
	FRAGMNETS_2 DivideMethod = 2
	FRAGMNETS_4 DivideMethod = 4
	FRAGMNETS_8 DivideMethod = 8
)

// 获得字节data的第n位
func getBit(data byte, n uint8) uint8 {
	if data&(1<<n) == (1 << n) {
		return 1
	} else {
		return 0
	}
}

// 把字节data的第n位置为flag
func setBit(data *byte, n uint8, flag uint8) {
	if flag == 1 {
		*data = (*data) | (1 << n)
	} else {
		*data = (*data) & ^(1 << n)
	}
}

// 将一个文件分成多个无加密的fragment,并组成顺序的列表(注意还不是组)
func GenerateDataFragmentList(filePath string, method DivideMethod) ([][]byte, error) {
	var err error
	plaintext, err := filetools.ReadFile(filePath)
	if err != nil {
		fmt.Println("无法读取待分片文件")
		return nil, err
	}
	var fragmentList [][]byte
	for i := 0; i < int(method); i++ {
		fragmentList = append(fragmentList, []byte{})
	}
	appendFragmentToList := func(wg *sync.WaitGroup, position []uint8, fragment *[]byte) {
		for i := range plaintext {
			var fragmentByte byte
			for _, bit := range position {
				flag := getBit(plaintext[i], bit)
				setBit(&fragmentByte, bit, flag)
			}
			*fragment = append(*fragment, fragmentByte)
		}
		wg.Done()
	}
	var wg sync.WaitGroup // 信号量
	switch method {
	case FRAGMNETS_2:
		wg.Add(2)
		go appendFragmentToList(&wg, []uint8{1, 3, 5, 7}, &fragmentList[0])
		go appendFragmentToList(&wg, []uint8{0, 2, 4, 6}, &fragmentList[1])
	case FRAGMNETS_4:
		wg.Add(4)
		go appendFragmentToList(&wg, []uint8{3, 7}, &fragmentList[0])
		go appendFragmentToList(&wg, []uint8{2, 6}, &fragmentList[1])
		go appendFragmentToList(&wg, []uint8{1, 5}, &fragmentList[2])
		go appendFragmentToList(&wg, []uint8{0, 4}, &fragmentList[3])
	case FRAGMNETS_8:
		wg.Add(8)
		go appendFragmentToList(&wg, []uint8{7}, &fragmentList[0])
		go appendFragmentToList(&wg, []uint8{6}, &fragmentList[1])
		go appendFragmentToList(&wg, []uint8{5}, &fragmentList[2])
		go appendFragmentToList(&wg, []uint8{4}, &fragmentList[3])
		go appendFragmentToList(&wg, []uint8{3}, &fragmentList[4])
		go appendFragmentToList(&wg, []uint8{2}, &fragmentList[5])
		go appendFragmentToList(&wg, []uint8{1}, &fragmentList[6])
		go appendFragmentToList(&wg, []uint8{0}, &fragmentList[7])
	default:
		panic("分片数量不合法")
	}
	wg.Wait()
	return fragmentList, err
}

// 将一个列表中的无加密的fragment还原成文件
func RestoreByFragmentList(filePath string, fragmentList [][]byte) error {
	var err error
	plaintext := make([]byte, len(fragmentList[0]))
	var method DivideMethod = DivideMethod(len(fragmentList))
	combineToPlaintext := func(wg *sync.WaitGroup, position []uint8, fragment *[]byte) {
		for j, fragmentByte := range *fragment {
			for _, bit := range position {
				flag := getBit(fragmentByte, bit)
				setBit(&plaintext[j], bit, flag)
			}
		}
		// wg.Done()
	}
	var wg sync.WaitGroup // 信号量
	switch method {
	case FRAGMNETS_2:
		combineToPlaintext(&wg, []uint8{1, 3, 5, 7}, &fragmentList[0])
		combineToPlaintext(&wg, []uint8{0, 2, 4, 6}, &fragmentList[1])
		// wg.Add(2)
	case FRAGMNETS_4:
		combineToPlaintext(&wg, []uint8{3, 7}, &fragmentList[0])
		combineToPlaintext(&wg, []uint8{2, 6}, &fragmentList[1])
		combineToPlaintext(&wg, []uint8{1, 5}, &fragmentList[2])
		combineToPlaintext(&wg, []uint8{0, 4}, &fragmentList[3])
		wg.Add(4)
		// wg.Add(2)
	case FRAGMNETS_8:
		combineToPlaintext(&wg, []uint8{7}, &fragmentList[0])
		combineToPlaintext(&wg, []uint8{6}, &fragmentList[1])
		combineToPlaintext(&wg, []uint8{5}, &fragmentList[2])
		combineToPlaintext(&wg, []uint8{4}, &fragmentList[3])
		combineToPlaintext(&wg, []uint8{3}, &fragmentList[4])
		combineToPlaintext(&wg, []uint8{2}, &fragmentList[5])
		combineToPlaintext(&wg, []uint8{1}, &fragmentList[6])
		combineToPlaintext(&wg, []uint8{0}, &fragmentList[7])
		// wg.Add(8)
	default:
		panic("分片数量不合法")
	}
	// wg.Wait()
	err = filetools.WriteFile(filePath, plaintext, 0777)
	return err
}

// 把fragment组成的列表转成均分的组
func ListToGroup(fragmentList [][]byte, groupNum int) (fragmentGroup [][][]byte) {
	maxNumInAGroup := int(math.Ceil(float64(len(fragmentList)) / float64(groupNum)))
	for i := 0; i < groupNum; i++ {
		fragmentGroup = append(fragmentGroup, [][]byte{})
	}
	for i, fragment := range fragmentList {
		row := i / maxNumInAGroup
		fragmentGroup[row] = append(fragmentGroup[row], fragment)
	}
	return fragmentGroup
}
